package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/go-service-broker/api/handlers"
	"github.com/pivotal-golang/lager"
)

type BrokerCredentials struct {
	Username string
	Password string
}

func auth(handler http.Handler, credentials BrokerCredentials) http.Handler {
	checkAuth := handlers.CheckAuth(
		credentials.Username,
		credentials.Password,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuth(w, r)
		handler.ServeHTTP(w, r)
	})
}

func New(serviceBroker ServiceBroker, brokerLogger lager.Logger, brokerCredentials BrokerCredentials) http.Handler {
	router := mux.NewRouter()

	// Catalog
	router.HandleFunc("/v2/catalog", func(w http.ResponseWriter, req *http.Request) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}

		json.NewEncoder(w).Encode(catalog)
	})

	// Provision
	router.HandleFunc("/v2/service_instances/{instance_id}", func(w http.ResponseWriter, req *http.Request) {
		serviceDetails := make(map[string]string)
		body, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body, &serviceDetails)

		vars := mux.Vars(req)
		instanceID := vars["instance_id"]
		err := serviceBroker.Provision(instanceID, serviceDetails)

		logger := brokerLogger.Session("provision", lager.Data{
			"instance-id":      instanceID,
			"instance-details": serviceDetails,
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
	}).Methods("PUT")

	// Deprovision
	router.HandleFunc("/v2/service_instances/{instance_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		instanceID := vars["instance_id"]
		logger := brokerLogger.Session("deprovision", lager.Data{
			"instance-id": instanceID,
		})
		err := serviceBroker.Deprovision(instanceID)
		if err != nil {
			logger.Error("instance-missing", err)
			w.WriteHeader(http.StatusGone)
		}

		json.NewEncoder(w).Encode(EmptyResponse{})
	}).Methods("DELETE")

	// Bind
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := brokerLogger.Session("bind", lager.Data{
			"instance-id": instanceID,
			"binding-id":  bindingID,
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
	}).Methods("PUT")

	// Unbind
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		instanceID := vars["instance_id"]
		bindingID := vars["binding_id"]

		logger := brokerLogger.Session("unbind", lager.Data{
			"instance-id": instanceID,
			"binding-id":  bindingID,
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
	}).Methods("DELETE")

	return auth(router, brokerCredentials)
}
