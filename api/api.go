package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/gorilla/mux"
	"github.com/martini-contrib/render"
	"github.com/pivotal-cf/go-service-broker/api/handlers"
	"github.com/pivotal-golang/lager"
)

func proxy(classicHandler *martini.ClassicMartini, newHandler http.Handler) http.Handler {
	auth := handlers.CheckAuth()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		method := r.Method
		parts := strings.Split(url[1:], "/")

		if strings.HasPrefix(url, "/v2/catalog") {
			auth(w, r)
			newHandler.ServeHTTP(w, r)
		} else if strings.HasPrefix(url, "/v2/service_instances") && method == "DELETE" && len(parts) == 3 {
			auth(w, r)
			newHandler.ServeHTTP(w, r)
		} else {
			classicHandler.ServeHTTP(w, r)
		}
	})
}

func New(serviceBroker ServiceBroker, httpLogger *log.Logger, brokerLogger lager.Logger) http.Handler {
	m := martini.Classic()
	m.Map(httpLogger)
	m.Handlers(
		handlers.CheckAuth(),
		render.Renderer(),
	)

	router := mux.NewRouter()

	// Catalog
	router.HandleFunc("/v2/catalog", func(w http.ResponseWriter, req *http.Request) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}

		json.NewEncoder(w).Encode(catalog)
	})

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
	})

	// Provision
	m.Put("/v2/service_instances/:instance_id", func(params martini.Params, r render.Render, req *http.Request) {
		serviceDetails := make(map[string]string)
		body, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body, &serviceDetails)

		instanceID := params["instance_id"]
		err := serviceBroker.Provision(instanceID, serviceDetails)

		logger := brokerLogger.Session("provision", lager.Data{
			"instance-id":      instanceID,
			"instance-details": serviceDetails,
		})

		if err != nil {
			switch err {
			case ErrInstanceAlreadyExists:
				logger.Error("instance-already-exists", err)
				r.JSON(409, EmptyResponse{})
			case ErrInstanceLimitMet:
				logger.Error("instance-limit-reached", err)
				r.JSON(500, ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error("unknown-error", err)

				r.JSON(500, ErrorResponse{
					Description: "an unexpected error occurred",
				})
			}

			return
		}

		r.JSON(201, ProvisioningResponse{})
	})

	// Bind
	m.Put("/v2/service_instances/:instance_id/service_bindings/:binding_id", func(params martini.Params, r render.Render) {
		instanceID := params["instance_id"]
		bindingID := params["binding_id"]

		logger := brokerLogger.Session("bind", lager.Data{
			"instance-id": instanceID,
			"binding-id":  bindingID,
		})
		credentials, err := serviceBroker.Bind(instanceID, bindingID)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error("instance-missing", err)

				r.JSON(404, ErrorResponse{
					Description: err.Error(),
				})
			case ErrBindingAlreadyExists:
				logger.Error("binding-already-exists", err)

				r.JSON(409, ErrorResponse{
					Description: err.Error(),
				})
			default:
				logger.Error("unknown-error", err)

				r.JSON(500, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		bindingResponse := BindingResponse{
			Credentials: credentials,
		}
		r.JSON(201, bindingResponse)
	})

	// Unbind
	m.Delete("/v2/service_instances/:instance_id/service_bindings/:binding_id", func(params martini.Params, r render.Render) {
		instanceID := params["instance_id"]
		bindingID := params["binding_id"]

		logger := brokerLogger.Session("unbind", lager.Data{
			"instance-id": instanceID,
			"binding-id":  bindingID,
		})

		err := serviceBroker.Unbind(instanceID, bindingID)

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				logger.Error("instance-missing", err)

				r.JSON(404, EmptyResponse{})
			case ErrBindingDoesNotExist:
				logger.Error("binding-missing", err)

				r.JSON(410, EmptyResponse{})
			default:
				logger.Error("unknown-error", err)

				r.JSON(500, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		r.JSON(200, EmptyResponse{})
	})

	return proxy(m, router)
}
