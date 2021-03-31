package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v7/middlewares"
	"github.com/pivotal-cf/brokerapi/v7/utils"
)

const (
	provisionLogKey = "provision"

	instanceDetailsLogKey = "instance-details"

	invalidServiceID = "invalid-service-id"
	invalidPlanID    = "invalid-plan-id"
)

func (h *APIHandler) Provision(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	logger := h.logger.Session(provisionLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	}, utils.DataForContext(req.Context(), middlewares.CorrelationIDKey))

	ctx := req.Context()
	originatingIdentity := fmt.Sprintf("%v", ctx.Value("requestIdentity"))

	var details domain.ProvisionDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(invalidServiceDetailsErrorKey, err)
		h.respond(w, http.StatusUnprocessableEntity, originatingIdentity, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	if details.ServiceID == "" {
		logger.Error(serviceIdMissingKey, serviceIdError)
		h.respond(w, http.StatusBadRequest, originatingIdentity, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		return
	}

	if details.PlanID == "" {
		logger.Error(planIdMissingKey, planIdError)
		h.respond(w, http.StatusBadRequest, originatingIdentity, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		return
	}

	valid := false
	services, _ := h.serviceBroker.Services(req.Context())
	for _, service := range services {
		if service.ID == details.ServiceID {
			req = req.WithContext(utils.AddServiceToContext(req.Context(), &service))
			valid = true
			break
		}
	}
	if !valid {
		logger.Error(invalidServiceID, invalidServiceIDError)
		h.respond(w, http.StatusBadRequest, originatingIdentity, apiresponses.ErrorResponse{
			Description: invalidServiceIDError.Error(),
		})
		return
	}

	valid = false
	for _, service := range services {
		for _, plan := range service.Plans {
			if plan.ID == details.PlanID {
				req = req.WithContext(utils.AddServicePlanToContext(req.Context(), &plan))
				valid = true
				break
			}
		}
	}
	if !valid {
		logger.Error(invalidPlanID, invalidPlanIDError)
		h.respond(w, http.StatusBadRequest, originatingIdentity, apiresponses.ErrorResponse{
			Description: invalidPlanIDError.Error(),
		})
		return
	}

	asyncAllowed := req.FormValue("accepts_incomplete") == "true"

	logger = logger.WithData(lager.Data{
		instanceDetailsLogKey: details,
	})

	provisionResponse, err := h.serviceBroker.Provision(req.Context(), instanceID, details, asyncAllowed)

	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(logger), originatingIdentity, err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, originatingIdentity, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	var metadata interface{}
	if !provisionResponse.Metadata.IsEmpty() {
		metadata = provisionResponse.Metadata
	}

	if provisionResponse.AlreadyExists {
		h.respond(w, http.StatusOK, originatingIdentity, apiresponses.ProvisioningResponse{
			DashboardURL: provisionResponse.DashboardURL,
			Metadata:     metadata,
		})
	} else if provisionResponse.IsAsync {
		h.respond(w, http.StatusAccepted, originatingIdentity, apiresponses.ProvisioningResponse{
			DashboardURL:  provisionResponse.DashboardURL,
			OperationData: provisionResponse.OperationData,
			Metadata:      metadata,
		})
	} else {
		h.respond(w, http.StatusCreated, originatingIdentity, apiresponses.ProvisioningResponse{
			DashboardURL: provisionResponse.DashboardURL,
			Metadata:     metadata,
		})
	}
}
