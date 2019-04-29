// Copyright (C) 2015-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brokerapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/handlers"
	"github.com/pivotal-cf/brokerapi/middlewares"
)

const (
	provisionLogKey            = "provision"
	deprovisionLogKey          = "deprovision"
	bindLogKey                 = "bind"
	getBindLogKey              = "getBinding"
	getInstanceLogKey          = "getInstance"
	unbindLogKey               = "unbind"
	updateLogKey               = "update"
	lastOperationLogKey        = "lastOperation"
	lastBindingOperationLogKey = "lastBindingOperation"

	instanceIDLogKey      = "instance-id"
	instanceDetailsLogKey = "instance-details"

	bindingIDLogKey               = "binding-id"
	invalidServiceDetailsErrorKey = "invalid-service-details"
	invalidBindDetailsErrorKey    = "invalid-bind-details"
	unknownErrorKey               = "unknown-error"
	apiVersionInvalidKey          = "broker-api-version-invalid"
	serviceIdMissingKey           = "service-id-missing"
	planIdMissingKey              = "plan-id-missing"
	invalidServiceID              = "invalid-service-id"
	invalidPlanID                 = "invalid-plan-id"
)

var (
	serviceIdError        = errors.New("service_id missing")
	planIdError           = errors.New("plan_id missing")
	invalidServiceIDError = errors.New("service-id not in the catalog")
	invalidPlanIDError    = errors.New("plan-id not in the catalog")
)

type BrokerCredentials struct {
	Username string
	Password string
}

func New(serviceBroker ServiceBroker, logger lager.Logger, brokerCredentials BrokerCredentials) http.Handler {
	router := mux.NewRouter()

	AttachRoutes(router, serviceBroker, logger)

	authMiddleware := auth.NewWrapper(brokerCredentials.Username, brokerCredentials.Password).Wrap
	apiVersionMiddleware := middlewares.APIVersionMiddleware{LoggerFactory: logger}

	router.Use(authMiddleware)
	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)

	return router
}

func AttachRoutes(router *mux.Router, serviceBroker ServiceBroker, logger lager.Logger) {
	handler := serviceBrokerHandler{serviceBroker: serviceBroker, logger: logger}

	apiHandler := handlers.APIHandler{serviceBroker, logger}
	router.HandleFunc("/v2/catalog", apiHandler.Catalog).Methods("GET")

	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.GetInstance).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Provision).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", handler.lastOperation).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Update).Methods("PATCH")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.unbind).Methods("DELETE")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", handler.lastBindingOperation).Methods("GET")
}

type serviceBrokerHandler struct {
	serviceBroker domain.ServiceBroker
	logger        lager.Logger
}

func (h serviceBrokerHandler) bind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	logger := h.logger.Session(bindLogKey, lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
	})

	version := getAPIVersion(req)
	asyncAllowed := false
	if version.Minor >= 14 {
		asyncAllowed = req.FormValue("accepts_incomplete") == "true"
	}

	var details domain.BindDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(invalidBindDetailsErrorKey, err)
		h.respond(w, http.StatusUnprocessableEntity, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	if details.ServiceID == "" {
		logger.Error(serviceIdMissingKey, serviceIdError)
		h.respond(w, http.StatusBadRequest, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		return
	}

	if details.PlanID == "" {
		logger.Error(planIdMissingKey, planIdError)
		h.respond(w, http.StatusBadRequest, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		return
	}

	binding, err := h.serviceBroker.Bind(req.Context(), instanceID, bindingID, details, asyncAllowed)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			statusCode := err.ValidatedStatusCode(logger)
			errorResponse := err.ErrorResponse()
			if err == apiresponses.ErrInstanceDoesNotExist {
				// work around ErrInstanceDoesNotExist having different pre-refactor behaviour to other actions
				errorResponse = apiresponses.ErrorResponse{
					Description: err.Error(),
				}
				statusCode = http.StatusNotFound
			}
			logger.Error(err.LoggerAction(), err)
			h.respond(w, statusCode, errorResponse)
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if binding.IsAsync {
		h.respond(w, http.StatusAccepted, apiresponses.AsyncBindResponse{
			OperationData: binding.OperationData,
		})
		return
	}

	if version.Minor == 8 || version.Minor == 9 {
		experimentalVols := []domain.ExperimentalVolumeMount{}

		for _, vol := range binding.VolumeMounts {
			experimentalConfig, err := json.Marshal(vol.Device.MountConfig)
			if err != nil {
				logger.Error(unknownErrorKey, err)
				h.respond(w, http.StatusInternalServerError, apiresponses.ErrorResponse{Description: err.Error()})
				return
			}

			experimentalVols = append(experimentalVols, domain.ExperimentalVolumeMount{
				ContainerPath: vol.ContainerDir,
				Mode:          vol.Mode,
				Private: domain.ExperimentalVolumeMountPrivate{
					Driver:  vol.Driver,
					GroupID: vol.Device.VolumeId,
					Config:  string(experimentalConfig),
				},
			})
		}

		experimentalBinding := apiresponses.ExperimentalVolumeMountBindingResponse{
			Credentials:     binding.Credentials,
			RouteServiceURL: binding.RouteServiceURL,
			SyslogDrainURL:  binding.SyslogDrainURL,
			VolumeMounts:    experimentalVols,
		}
		h.respond(w, http.StatusCreated, experimentalBinding)
		return
	}

	h.respond(w, http.StatusCreated, apiresponses.BindingResponse{
		Credentials:     binding.Credentials,
		SyslogDrainURL:  binding.SyslogDrainURL,
		RouteServiceURL: binding.RouteServiceURL,
		VolumeMounts:    binding.VolumeMounts,
	})
}

func (h serviceBrokerHandler) unbind(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]

	logger := h.logger.Session(unbindLogKey, lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
	})

	version := getAPIVersion(req)
	asyncAllowed := req.FormValue("accepts_incomplete") == "true"
	if asyncAllowed && version.Minor < 14 {
		err := errors.New("async unbinding only supported from OSB version 2.14 and up")
		h.respond(w, http.StatusUnprocessableEntity, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(apiVersionInvalidKey, err)
		return
	}
	details := domain.UnbindDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
	}

	if details.ServiceID == "" {
		h.respond(w, http.StatusBadRequest, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		logger.Error(serviceIdMissingKey, serviceIdError)
		return
	}

	if details.PlanID == "" {
		h.respond(w, http.StatusBadRequest, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		logger.Error(planIdMissingKey, planIdError)
		return
	}

	unbindResponse, err := h.serviceBroker.Unbind(req.Context(), instanceID, bindingID, details, asyncAllowed)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(logger), err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if unbindResponse.IsAsync {
		h.respond(w, http.StatusAccepted, apiresponses.UnbindResponse{
			OperationData: unbindResponse.OperationData,
		})
	} else {
		h.respond(w, http.StatusOK, apiresponses.EmptyResponse{})
	}

}

func (h serviceBrokerHandler) lastBindingOperation(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	bindingID := vars["binding_id"]
	pollDetails := domain.PollDetails{
		PlanID:        req.FormValue("plan_id"),
		ServiceID:     req.FormValue("service_id"),
		OperationData: req.FormValue("operation"),
	}

	logger := h.logger.Session(lastBindingOperationLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	})

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get binding endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(apiVersionInvalidKey, err)
		return
	}

	logger.Info("starting-check-for-binding-operation")

	lastOperation, err := h.serviceBroker.LastBindingOperation(req.Context(), instanceID, bindingID, pollDetails)

	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(logger), err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	logger.WithData(lager.Data{"state": lastOperation.State}).Info("done-check-for-binding-operation")

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
		Description: lastOperation.Description,
	}
	h.respond(w, http.StatusOK, lastOperationResponse)
}

func (h serviceBrokerHandler) lastOperation(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]
	pollDetails := domain.PollDetails{
		PlanID:        req.FormValue("plan_id"),
		ServiceID:     req.FormValue("service_id"),
		OperationData: req.FormValue("operation"),
	}

	logger := h.logger.Session(lastOperationLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	})

	logger.Info("starting-check-for-operation")

	lastOperation, err := h.serviceBroker.LastOperation(req.Context(), instanceID, pollDetails)

	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(logger), err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	logger.WithData(lager.Data{"state": lastOperation.State}).Info("done-check-for-operation")

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
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

type brokerVersion struct {
	Major int
	Minor int
}

func getAPIVersion(req *http.Request) brokerVersion {
	var version brokerVersion
	apiVersion := req.Header.Get("X-Broker-API-Version")

	fmt.Sscanf(apiVersion, "%d.%d", &version.Major, &version.Minor)

	return version
}
