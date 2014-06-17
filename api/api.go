package api

import (
	"fmt"
	"log"

	"github.com/cloudfoundry/gosteno"
	"github.com/codegangsta/martini"
	"github.com/martini-contrib/render"
)

func New(serviceBroker ServiceBroker, httpLogger *log.Logger, brokerLogger *gosteno.Logger) *martini.ClassicMartini {
	m := martini.Classic()
	m.Map(httpLogger)
	m.Use(render.Renderer())

	// Catalog
	m.Get("/v2/catalog", func(r render.Render) {
		catalog := CatalogResponse{
			Services: serviceBroker.Services(),
		}
		r.JSON(200, catalog)
	})

	// Provision
	m.Put("/v2/service_instances/:instance_id", func(params martini.Params, r render.Render) {
		err := serviceBroker.Provision(params["instance_id"])

		if err != nil {
			switch err {
			case ErrInstanceAlreadyExists:
				errorLog := fmt.Sprintf("Provisioning error: instance %s already exists", params["instance_id"])
				brokerLogger.Warn(errorLog)
				r.JSON(409, EmptyResponse{})
			case ErrInstanceLimitMet:
				brokerLogger.Warn("Provisioning error: instance limit for this service has been reached")
				r.JSON(500, ErrorResponse{
					Description: err.Error(),
				})
			default:
				errorLog := fmt.Sprintf("Provisioning error: %s", err)
				brokerLogger.Warn(errorLog)

				r.JSON(500, ErrorResponse{
					Description: "an unexpected error occurred",
				})
			}

			return
		}

		r.JSON(201, ProvisioningResponse{})
	})

	// Deprovision
	m.Delete("/v2/service_instances/:instance_id", func(params martini.Params, r render.Render) {
		err := serviceBroker.Deprovision(params["instance_id"])
		if err != nil {
			errorLog := fmt.Sprintf("Deprovisioning error: instance %s does not exist", params["instance_id"])
			brokerLogger.Warn(errorLog)
			r.JSON(410, EmptyResponse{})
			return
		}
		r.JSON(200, EmptyResponse{})
	})

	// Bind
	m.Put("/v2/service_instances/:instance_id/service_bindings/:binding_id", func(params martini.Params, r render.Render) {
		credentials, err := serviceBroker.Bind(params["instance_id"], params["binding_id"])

		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				errorLog := fmt.Sprintf("Binding error: instance %s does not exist", params["instance_id"])
				brokerLogger.Warn(errorLog)
				r.JSON(404, ErrorResponse{
					Description: err.Error(),
				})
			case ErrBindingAlreadyExists:
				errorLog := fmt.Sprintf("Binding error: binding already exists")
				brokerLogger.Warn(errorLog)
				r.JSON(409, ErrorResponse{
					Description: err.Error(),
				})
			default:
				errorLog := fmt.Sprintf(err.Error())
				brokerLogger.Error(errorLog)

				r.JSON(500, ErrorResponse{
					Description: ErrOtherInternal.Error(),
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
		err := serviceBroker.Unbind(params["instance_id"], params["binding_id"])
		if err != nil {
			switch err {
			case ErrInstanceDoesNotExist:
				errorLog := fmt.Sprintf("Unbinding error: instance %s does not exist", params["instance_id"])
				brokerLogger.Warn(errorLog)
				r.JSON(404, EmptyResponse{})
			case ErrBindingDoesNotExist:
				errorLog := fmt.Sprintf("Unbinding error: binding %s does not exist", params["binding_id"])
				brokerLogger.Warn(errorLog)
				r.JSON(410, EmptyResponse{})
			default:
				r.JSON(500, ErrorResponse{
					Description: err.Error(),
				})
			}
			return
		}

		r.JSON(200, EmptyResponse{})
	})

	return m
}
