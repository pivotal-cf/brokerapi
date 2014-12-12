package api

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf/go-service-broker/api/handlers"
	"github.com/pivotal-golang/lager"
)

const provisionLogKey = "provision"
const deprovisionLogKey = "deprovision"
const bindLogKey = "bind"
const unbindLogKey = "unbind"

const instanceIDLogKey = "instance-id"
const instanceDetailsLogKey = "instance-details"
const bindingIDLogKey = "binding-id"

const invalidServiceDetailsErrorKey = "invalid-service-details"
const instanceLimitReachedErrorKey = "instance-limit-reached"
const instanceAlreadyExistsErrorKey = "instance-already-exists"
const bindingAlreadyExistsErrorKey = "binding-already-exists"
const instanceMissingErrorKey = "instance-missing"
const bindingMissingErrorKey = "binding-missing"
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

	router.Put("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", bind(serviceBroker, router, logger))
	router.Delete("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", unbind(serviceBroker, router, logger))

	return auth(router, brokerCredentials)
}

func auth(router httpRouter, credentials BrokerCredentials) http.Handler {
	checkAuth := handlers.CheckAuth(
		credentials.Username,
		credentials.Password,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuth(w, r)
		router.ServeHTTP(w, r)
	})
}

func catalog(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}

		json.NewEncoder(w).Encode(catalog)
	}
}

func provision(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var serviceDetails ServiceDetails
		err := json.NewDecoder(req.Body).Decode(&serviceDetails)

		vars := router.Vars(req)
		instanceID := vars["instance_id"]

		if err != nil {
			w.WriteHeader(statusUnprocessableEntity)

			logger := logger.Session(provisionLogKey, lager.Data{
				instanceIDLogKey:      instanceID,
				instanceDetailsLogKey: nil,
			})

			logger.Error(invalidServiceDetailsErrorKey, err)
			return
		}

		err = serviceBroker.Provision(instanceID, serviceDetails)

		logger := logger.Session(provisionLogKey, lager.Data{
			instanceIDLogKey:      instanceID,
			instanceDetailsLogKey: serviceDetails,
		})

		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceAlreadyExists:
				logger.Error(instanceAlreadyExistsErrorKey, err)
				w.WriteHeader(http.StatusConflict)
				encoder.Encode(EmptyResponse{})
			case ErrInstanceLimitMet:
				logger.Error(instanceLimitReachedErrorKey, err)
				w.WriteHeader(http.StatusInternalServerError)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error(unknownErrorKey, err)
				w.WriteHeader(http.StatusInternalServerError)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			}

			return
		}

		w.WriteHeader(http.StatusCreated)
		encoder.Encode(ProvisioningResponse{})
	}
}

func deprovision(serviceBroker ServiceBroker, router httpRouter, logger lager.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := router.Vars(req)
		instanceID := vars["instance_id"]
		logger := logger.Session(deprovisionLogKey, lager.Data{
			instanceIDLogKey: instanceID,
		})
		encoder := json.NewEncoder(w)

		err := serviceBroker.Deprovision(instanceID)
		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				w.WriteHeader(http.StatusGone)
				encoder.Encode(EmptyResponse{})
			default:
				logger.Error(unknownErrorKey, err)
				w.WriteHeader(http.StatusInternalServerError)
				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			}

			return
		}

		encoder.Encode(EmptyResponse{})
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
		credentials, err := serviceBroker.Bind(instanceID, bindingID)
		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				w.WriteHeader(http.StatusNotFound)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			case ErrBindingAlreadyExists:
				logger.Error(bindingAlreadyExistsErrorKey, err)
				w.WriteHeader(http.StatusConflict)

				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error(unknownErrorKey, err)
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

		err := serviceBroker.Unbind(instanceID, bindingID)
		encoder := json.NewEncoder(w)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error(instanceMissingErrorKey, err)
				w.WriteHeader(http.StatusNotFound)
				encoder.Encode(EmptyResponse{})
			case ErrBindingDoesNotExist:
				logger.Error(bindingMissingErrorKey, err)
				w.WriteHeader(http.StatusGone)
				encoder.Encode(EmptyResponse{})
			default:
				logger.Error(unknownErrorKey, err)
				w.WriteHeader(http.StatusInternalServerError)
				encoder.Encode(ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		encoder.Encode(EmptyResponse{})
	}
}
