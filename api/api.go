package api

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf/go-service-broker/api/handlers"
	"github.com/pivotal-golang/lager"
)

const instance_id_log_key = "instance-id"
const instance_details_log_key = "instance-details"
const StatusUnprocessableEntity = 422

type BrokerCredentials struct {
	Username string
	Password string
}

func auth(router HttpRouter, credentials BrokerCredentials) http.Handler {
	checkAuth := handlers.CheckAuth(
		credentials.Username,
		credentials.Password,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuth(w, r)
		router.ServeHTTP(w, r)
	})
}

func New(serviceBroker ServiceBroker, brokerLogger lager.Logger, brokerCredentials BrokerCredentials) http.Handler {
	router := NewHttpRouter()

	// Catalog
	router.Get("/v2/catalog", func(w http.ResponseWriter, req *http.Request) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}

		json.NewEncoder(w).Encode(catalog)
	})

	// Provision
	router.Put("/v2/service_instances/{instance_id}", func(w http.ResponseWriter, req *http.Request) {
		var serviceDetails ServiceDetails
		err := json.NewDecoder(req.Body).Decode(&serviceDetails)

		vars := router.Vars(req)
		instanceID := vars["instance_id"]

		if err != nil {
			w.WriteHeader(StatusUnprocessableEntity)

			logger := brokerLogger.Session("provision", lager.Data{
				instance_id_log_key:      instanceID,
				instance_details_log_key: nil,
			})

			logger.Error("invalid-service-details", err)
			return
		}

		err = serviceBroker.Provision(instanceID, serviceDetails)

		logger := brokerLogger.Session("provision", lager.Data{
			instance_id_log_key:      instanceID,
			instance_details_log_key: serviceDetails,
		})

		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceAlreadyExists:
				logger.Error("instance-already-exists", err)
				w.WriteHeader(http.StatusConflict)
				encoder.Encode(EmptyResponse{})
			case ErrInstanceLimitMet:
				logger.Error("instance-limit-reached", err)
				w.WriteHeader(http.StatusInternalServerError)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error("unknown-error", err)
				w.WriteHeader(http.StatusInternalServerError)

				encoder.Encode(ErrorResponse{
					Description: "an unexpected error occurred",
				})
			}

			return
		}

		w.WriteHeader(http.StatusCreated)
		encoder.Encode(ProvisioningResponse{})
	})

	// Deprovision
	router.Delete("/v2/service_instances/{instance_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		logger := brokerLogger.Session("deprovision", lager.Data{
			instance_id_log_key: instanceID,
		})
		err := serviceBroker.Deprovision(instanceID)
		if err != nil {
			logger.Error("instance-missing", err)
			w.WriteHeader(http.StatusGone)
		}

		json.NewEncoder(w).Encode(EmptyResponse{})
	})

	// Bind
	router.Put("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := brokerLogger.Session("bind", lager.Data{
			instance_id_log_key: instanceID,
			"binding-id":        bindingID,
		})
		credentials, err := serviceBroker.Bind(instanceID, bindingID)
		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error("instance-missing", err)
				w.WriteHeader(http.StatusNotFound)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			case ErrBindingAlreadyExists:
				logger.Error("binding-already-exists", err)
				w.WriteHeader(http.StatusConflict)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error("unknown-error", err)
				w.WriteHeader(http.StatusInternalServerError)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		bindingResponse := BindingResponse{
			Credentials: credentials,
		}

		w.WriteHeader(http.StatusCreated)
		encoder.Encode(bindingResponse)
	})

	// Unbind
	router.Delete("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := brokerLogger.Session("unbind", lager.Data{
			instance_id_log_key: instanceID,
			"binding-id":        bindingID,
		})

		err := serviceBroker.Unbind(instanceID, bindingID)
		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error("instance-missing", err)
				w.WriteHeader(http.StatusNotFound)
				encoder.Encode(EmptyResponse{})
			case ErrBindingDoesNotExist:
				logger.Error("binding-missing", err)
				w.WriteHeader(http.StatusGone)
				encoder.Encode(EmptyResponse{})
			default:
				logger.Error("unknown-error", err)
				w.WriteHeader(http.StatusInternalServerError)
				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		encoder.Encode(EmptyResponse{})
	})

	return auth(router, brokerCredentials)
}
