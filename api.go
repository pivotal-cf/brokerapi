package brokerapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-golang/lager"
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

const statusUnprocessableEntity = 422

type BrokerCredentials struct {
	Username string
	Password string
}

func New(serviceBroker ServiceBroker, logger lager.Logger, brokerCredentials BrokerCredentials) http.Handler {
	router := newHttpRouter()

	router.Get("/v2/catalog", catalog(serviceBroker, router, logger))

	router.Put("/v2/service_instances/{instance_id}", provision(serviceBroker, router, logger))
	router.Delete("/v2/service_instances/{instance_id}", deprovision(serviceBroker, router, logger))
	router.Get("/v2/service_instances/{instance_id}/last_operation", lastOperation(serviceBroker, router, logger))
	router.Patch("/v2/service_instances/{instance_id}", update(serviceBroker, router, logger))

	router.Put("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", bind(serviceBroker, router, logger))
	router.Delete("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", unbind(serviceBroker, router, logger))

	return wrapAuth(router, brokerCredentials)
}

func wrapAuth(router httpRouter, credentials BrokerCredentials) http.Handler {
	return auth.NewWrapper(credentials.Username, credentials.Password).Wrap(router)
}

func catalog(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}

		respond(w, http.StatusOK, catalog)
	}
}

func provision(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]

		logger := logger.Session(provisionLogKey, lager.Data{
			instanceIDLogKey: instanceID,
		})

		var details ProvisionDetails
		if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
			logger.Error(invalidServiceDetailsErrorKey, err)
			respond(w, statusUnprocessableEntity, ErrorResponse{
				Description: err.Error(),
			})
			return
		}

		acceptsIncompleteFlag, _ := strconv.ParseBool(req.URL.Query().Get("accepts_incomplete"))

		logger = logger.WithData(lager.Data{
			instanceDetailsLogKey: details,
		})

		provisionResponse, err := serviceBroker.Provision(instanceID, details, acceptsIncompleteFlag)

		if err != nil {
			switch err {
			case ErrInstanceAlreadyExists:
				logger.Error(instanceAlreadyExistsErrorKey, err)
				respond(w, http.StatusConflict, EmptyResponse{})
			case ErrInstanceLimitMet:
				logger.Error(instanceLimitReachedErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
			case ErrAsyncRequired:
				logger.Error(asyncRequiredKey, err)
				respond(w, 422, ErrorResponse{
					Error:       "AsyncRequired",
					Description: err.Error(),
				})
			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		if provisionResponse.IsAsync {
			respond(w, http.StatusAccepted, ProvisioningResponse{
				DashboardURL: provisionResponse.DashboardURL,
			})
		} else {
			respond(w, http.StatusCreated, ProvisioningResponse{
				DashboardURL: provisionResponse.DashboardURL,
			})
		}
	}
}

func update(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]

		var details UpdateDetails
		if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
			logger.Error(invalidServiceDetailsErrorKey, err)
			respond(w, statusUnprocessableEntity, ErrorResponse{
				Description: err.Error(),
			})
			return
		}

		acceptsIncompleteFlag, _ := strconv.ParseBool(req.URL.Query().Get("accepts_incomplete"))

		isAsync, err := serviceBroker.Update(instanceID, details, acceptsIncompleteFlag)
		if err != nil {
			switch err {
			case ErrAsyncRequired:
				logger.Error(asyncRequiredKey, err)
				respond(w, 422, ErrorResponse{
					Error:       "AsyncRequired",
					Description: err.Error(),
				})
				return

			case ErrPlanChangeNotSupported:
				logger.Error(planChangeNotSupportedKey, err)
				respond(w, 422, ErrorResponse{
					Error:       "PlanChangeNotSupported",
					Description: err.Error(),
				})
				return

			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
				return
			}
		}

		statusCode := http.StatusOK
		if isAsync {
			statusCode = http.StatusAccepted
		}
		respond(w, statusCode, struct{}{})
	}
}

func deprovision(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		logger := logger.Session(deprovisionLogKey, lager.Data{
			instanceIDLogKey: instanceID,
		})

		details := DeprovisionDetails{
			PlanID:    req.FormValue("plan_id"),
			ServiceID: req.FormValue("service_id"),
		}
		asyncAllowed := req.FormValue("accepts_incomplete") == "true"

		isAsync, err := serviceBroker.Deprovision(instanceID, details, asyncAllowed)
		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				respond(w, http.StatusGone, EmptyResponse{})
			case ErrAsyncRequired:
				logger.Error(asyncRequiredKey, err)
				respond(w, 422, EmptyResponse{})
			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		if isAsync {
			respond(w, http.StatusAccepted, EmptyResponse{})
		} else {
			respond(w, http.StatusOK, EmptyResponse{})
		}
	}
}

func bind(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := logger.Session(bindLogKey, lager.Data{
			instanceIDLogKey: instanceID,
			bindingIDLogKey:  bindingID,
		})

		var details BindDetails
		if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
			logger.Error(invalidBindDetailsErrorKey, err)
			respond(w, statusUnprocessableEntity, ErrorResponse{
				Description: err.Error(),
			})
			return
		}

		binding, err := serviceBroker.Bind(instanceID, bindingID, details)
		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				respond(w, http.StatusNotFound, ErrorResponse{
					Description: err.Error(),
				})
			case ErrBindingAlreadyExists:
				logger.Error(bindingAlreadyExistsErrorKey, err)
				respond(w, http.StatusConflict, ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		respond(w, http.StatusCreated, binding)
	}
}

func unbind(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := logger.Session(unbindLogKey, lager.Data{
			instanceIDLogKey: instanceID,
			bindingIDLogKey:  bindingID,
		})

		details := UnbindDetails{
			PlanID:    req.FormValue("plan_id"),
			ServiceID: req.FormValue("service_id"),
		}

		if err := serviceBroker.Unbind(instanceID, bindingID, details); err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				respond(w, http.StatusNotFound, EmptyResponse{})
			case ErrBindingDoesNotExist:
				logger.Error(bindingMissingErrorKey, err)
				respond(w, http.StatusGone, EmptyResponse{})
			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		respond(w, http.StatusOK, EmptyResponse{})
	}
}

func respond(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
		fmt.Printf("response being attempted %d %#v\n", status, response)
		fmt.Println(err)
	}
}

func lastOperation(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]

		logger := logger.Session(lastOperationLogKey, lager.Data{
			instanceIDLogKey: instanceID,
		})

		logger.Info("starting-check-for-operation")

		lastOperation, err := serviceBroker.LastOperation(instanceID)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				respond(w, http.StatusNotFound, ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error(unknownErrorKey, err)
				respond(w, http.StatusInternalServerError, ErrorResponse{
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

		respond(w, http.StatusOK, lastOperationResponse)
	}
}
