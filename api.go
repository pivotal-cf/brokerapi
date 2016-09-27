package brokerapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/auth"
	"strings"
)

const provisionLogKey = "provision"
const deprovisionLogKey = "deprovision"
const bindLogKey = "bind"
const unbindLogKey = "unbind"
const lastOperationLogKey = "lastOperation"

const instanceIDLogKey = "instance-id"
const instanceDetailsLogKey = "instance-details"
const bindingIDLogKey = "binding-id"

const invalidServiceDetailsErrorKey = "invalid-service-details"
const invalidBindDetailsErrorKey = "invalid-bind-details"
const invalidUnbindDetailsErrorKey = "invalid-unbind-details"
const invalidDeprovisionDetailsErrorKey = "invalid-deprovision-details"
const instanceLimitReachedErrorKey = "instance-limit-reached"
const instanceAlreadyExistsErrorKey = "instance-already-exists"
const bindingAlreadyExistsErrorKey = "binding-already-exists"
const instanceMissingErrorKey = "instance-missing"
const bindingMissingErrorKey = "binding-missing"
const asyncRequiredKey = "async-required"
const planChangeNotSupportedKey = "plan-change-not-supported"
const unknownErrorKey = "unknown-error"
const invalidRawParamsKey = "invalid-raw-params"
const appGuidNotProvidedErrorKey = "app-guid-not-provided"

const statusUnprocessableEntity = 422

type BrokerCredentials struct {
	Username string
	Password string
}

func New(serviceBroker ServiceBroker, logger lager.Logger, brokerCredentials BrokerCredentials) http.Handler {
	router := mux.NewRouter()
	AttachRoutes(router, serviceBroker, logger)
	return auth.NewWrapper(brokerCredentials.Username, brokerCredentials.Password).Wrap(router)
}

func AttachRoutes(router *mux.Router, serviceBroker ServiceBroker, logger lager.Logger) {
	handler := serviceBrokerHandler{serviceBroker: serviceBroker, logger: logger}
	router.HandleFunc("/v2/catalog", handler.catalog).Methods("GET")

	router.HandleFunc("/v2/service_instances/{instance_id}", handler.provision).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", handler.deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", handler.lastOperation).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", handler.update).Methods("PATCH")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.unbind).Methods("DELETE")
}

type serviceBrokerHandler struct {
	serviceBroker ServiceBroker
	logger        lager.Logger
}

func (h serviceBrokerHandler) catalog(w http.ResponseWriter, req *http.Request) {
	catalog := CatalogResponse{
		Services: h.serviceBroker.Services(),
	}

	h.respond(w, http.StatusOK, catalog)
}

func (h serviceBrokerHandler) provision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	logger := h.logger.Session(provisionLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	})

	var details ProvisionDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(invalidServiceDetailsErrorKey, err)
		h.respond(w, statusUnprocessableEntity, ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	acceptsIncompleteFlag, _ := strconv.ParseBool(req.URL.Query().Get("accepts_incomplete"))

	logger = logger.WithData(lager.Data{
		instanceDetailsLogKey: details,
	})

	provisionResponse, err := h.serviceBroker.Provision(instanceID, details, acceptsIncompleteFlag)

	if err != nil {
		switch err {
		case ErrRawParamsInvalid:
			logger.Error(invalidRawParamsKey, err)
			h.respond(w, 422, ErrorResponse{
				Description: err.Error(),
			})
		case ErrInstanceAlreadyExists:
			logger.Error(instanceAlreadyExistsErrorKey, err)
			h.respond(w, http.StatusConflict, EmptyResponse{})
		case ErrInstanceLimitMet:
			logger.Error(instanceLimitReachedErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		case ErrAsyncRequired:
			logger.Error(asyncRequiredKey, err)
			h.respond(w, 422, ErrorResponse{
				Error:       "AsyncRequired",
				Description: err.Error(),
			})
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if provisionResponse.IsAsync {
		h.respond(w, http.StatusAccepted, ProvisioningResponse{
			DashboardURL:  provisionResponse.DashboardURL,
			OperationData: provisionResponse.OperationData,
		})
	} else {
		h.respond(w, http.StatusCreated, ProvisioningResponse{
			DashboardURL: provisionResponse.DashboardURL,
		})
	}
}

func (h serviceBrokerHandler) update(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	var details UpdateDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		h.logger.Error(invalidServiceDetailsErrorKey, err)
		h.respond(w, statusUnprocessableEntity, ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	acceptsIncompleteFlag, _ := strconv.ParseBool(req.URL.Query().Get("accepts_incomplete"))

	updateServiceSpec, err := h.serviceBroker.Update(instanceID, details, acceptsIncompleteFlag)
	if err != nil {
		switch err {
		case ErrAsyncRequired:
			h.logger.Error(asyncRequiredKey, err)
			h.respond(w, 422, ErrorResponse{
				Error:       "AsyncRequired",
				Description: err.Error(),
			})
			return

		case ErrPlanChangeNotSupported:
			h.logger.Error(planChangeNotSupportedKey, err)
			h.respond(w, 422, ErrorResponse{
				Error:       "PlanChangeNotSupported",
				Description: err.Error(),
			})
			return

		default:
			h.logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
			return
		}
	}

	statusCode := http.StatusOK
	if updateServiceSpec.IsAsync {
		statusCode = http.StatusAccepted
	}
	h.respond(w, statusCode, UpdateResponse{OperationData: updateServiceSpec.OperationData})
}

func (h serviceBrokerHandler) deprovision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	logger := h.logger.Session(deprovisionLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	})

	details := DeprovisionDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
	}
	asyncAllowed := req.FormValue("accepts_incomplete") == "true"

	deprovisionSpec, err := h.serviceBroker.Deprovision(instanceID, details, asyncAllowed)
	if err != nil {
		switch err {
		case ErrInstanceDoesNotExist:
			logger.Error(instanceMissingErrorKey, err)
			h.respond(w, http.StatusGone, EmptyResponse{})
		case ErrAsyncRequired:
			logger.Error(asyncRequiredKey, err)
			h.respond(w, 422, ErrorResponse{
				Error:       "AsyncRequired",
				Description: err.Error(),
			})
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if deprovisionSpec.IsAsync {
		h.respond(w, http.StatusAccepted, DeprovisionResponse{OperationData: deprovisionSpec.OperationData})
	} else {
		h.respond(w, http.StatusOK, EmptyResponse{})
	}
}

func VersionCompare(lhs, rhs string) int {
	lhsParts := strings.Split(lhs, ".")
	rhsParts := strings.Split(rhs, ".")

	cmpLen := len(lhsParts)
	if len(rhsParts) < cmpLen {
		cmpLen = len(rhsParts)
	}
	for i := 0; i < cmpLen; i++ {
		lpart, _ := strconv.Atoi(lhsParts[i])
		rpart, _ := strconv.Atoi(rhsParts[i])
		if lpart > rpart {
			return 1
		} else if lpart < rpart {
			return -1
		}
	}

	// if we get here then the slots where both sides have a version part are both the same, so whichever side still has
	// stuff is the greater version
	if len(lhsParts) > len(rhsParts) {
		return 1
	}
	if len(lhsParts) < len(rhsParts) {
		return -1
	}
	return 0
}

func (h serviceBrokerHandler) bind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	logger := h.logger.Session(bindLogKey, lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
	})

	var details BindDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(invalidBindDetailsErrorKey, err)
		h.respond(w, statusUnprocessableEntity, ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	values, ok := req.Header["X-Broker-Api-Version"]
	if !ok && len(values) == 0 {
		// TODO--decide how we should handle this case
		values = []string{"2.10"}
	}

	if VersionCompare(values[0], "2.8") < 0 {
		err := errors.New("API version " + values[0] + " is not supported")
		logger.Error("unsupported-version", err)
		h.respond(w, http.StatusNotImplemented, ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	binding, err := h.serviceBroker.Bind(instanceID, bindingID, details)
	if err != nil {
		switch err {
		case ErrInstanceDoesNotExist:
			logger.Error(instanceMissingErrorKey, err)
			h.respond(w, http.StatusNotFound, ErrorResponse{
				Description: err.Error(),
			})
		case ErrBindingAlreadyExists:
			logger.Error(bindingAlreadyExistsErrorKey, err)
			h.respond(w, http.StatusConflict, ErrorResponse{
				Description: err.Error(),
			})
		case ErrAppGuidNotProvided:
			logger.Error(appGuidNotProvidedErrorKey, err)
			h.respond(w, statusUnprocessableEntity, ErrorResponse{
				Description: err.Error(),
			})
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if VersionCompare(values[0], "2.8") == 0 || VersionCompare(values[0], "2.9") == 0 {
		// convert 2.10 binding into something that a 2.9 client can understand
		newBinding := V2_9Binding{
			Credentials:     binding.Credentials,
			SyslogDrainURL:  binding.SyslogDrainURL,
			RouteServiceURL: binding.RouteServiceURL,
			VolumeMounts:    []V2_9VolumeMount{},
		}

		for _, mount := range binding.VolumeMounts {
			config, _ := json.Marshal(mount.Device.MountConfig)
			newBinding.VolumeMounts = append(newBinding.VolumeMounts, V2_9VolumeMount{
				ContainerPath: mount.ContainerDir,
				Mode:          mount.Mode,
				Private: V2_9VolumeMountPrivate{
					Driver:  mount.Driver,
					GroupId: mount.Device.VolumeId,
					Config:  string(config),
				},
			})
		}

		h.respond(w, http.StatusCreated, newBinding)
	} else {
		h.respond(w, http.StatusCreated, binding)
	}
}

func (h serviceBrokerHandler) unbind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	logger := h.logger.Session(unbindLogKey, lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
	})

	details := UnbindDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
	}

	if err := h.serviceBroker.Unbind(instanceID, bindingID, details); err != nil {
		switch err {
		case ErrInstanceDoesNotExist:
			logger.Error(instanceMissingErrorKey, err)
			h.respond(w, http.StatusGone, EmptyResponse{})
		case ErrBindingDoesNotExist:
			logger.Error(bindingMissingErrorKey, err)
			h.respond(w, http.StatusGone, EmptyResponse{})
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	h.respond(w, http.StatusOK, EmptyResponse{})
}

func (h serviceBrokerHandler) lastOperation(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	operationData := req.FormValue("operation")

	logger := h.logger.Session(lastOperationLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	})

	logger.Info("starting-check-for-operation")

	lastOperation, err := h.serviceBroker.LastOperation(instanceID, operationData)

	if err != nil {
		switch err {
		case ErrInstanceDoesNotExist:
			logger.Error(instanceMissingErrorKey, err)
			h.respond(w, http.StatusNotFound, ErrorResponse{
				Description: err.Error(),
			})
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, ErrorResponse{
				Description: err.Error(),
			})
		}

		return
	}

	logger.WithData(lager.Data{"state": lastOperation.State}).Info("done-check-for-operation")

	lastOperationResponse := LastOperationResponse{
		State:       string(lastOperation.State),
		Description: lastOperation.Description,
	}

	h.respond(w, http.StatusOK, lastOperationResponse)
}

func (h serviceBrokerHandler) respond(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		h.logger.Error("encoding response", err, lager.Data{"status": status, "response": response})
	}
}
