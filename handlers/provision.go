package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v10/internal/logutil"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
	"github.com/pivotal-cf/brokerapi/v10/utils"
)

const (
	provisionLogKey = "provision"

	instanceDetailsLogKey = "instance-details"

	invalidServiceID = "invalid-service-id"
	invalidPlanID    = "invalid-plan-id"
)

func (h *APIHandler) Provision(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	var details domain.ProvisionDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(logutil.Join(provisionLogKey, invalidServiceDetailsErrorKey), logutil.Error(err))
		h.respond(w, http.StatusUnprocessableEntity, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	if details.ServiceID == "" {
		logger.Error(logutil.Join(provisionLogKey, serviceIdMissingKey), logutil.Error(serviceIdError))
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		return
	}

	if details.PlanID == "" {
		logger.Error(logutil.Join(provisionLogKey, planIdMissingKey), logutil.Error(planIdError))
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
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
		logger.Error(logutil.Join(provisionLogKey, invalidServiceID), logutil.Error(invalidServiceIDError))
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
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
		logger.Error(logutil.Join(provisionLogKey, invalidPlanID), logutil.Error(invalidPlanIDError))
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: invalidPlanIDError.Error(),
		})
		return
	}

	asyncAllowed := req.FormValue("accepts_incomplete") == "true"

	logger = logger.With(slog.Any(instanceDetailsLogKey, details))

	provisionResponse, err := h.serviceBroker.Provision(req.Context(), instanceID, details, asyncAllowed)

	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(provisionLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(provisionLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(provisionLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
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
		h.respond(w, http.StatusOK, requestId, apiresponses.ProvisioningResponse{
			DashboardURL: provisionResponse.DashboardURL,
			Metadata:     metadata,
		})
	} else if provisionResponse.IsAsync {
		h.respond(w, http.StatusAccepted, requestId, apiresponses.ProvisioningResponse{
			DashboardURL:  provisionResponse.DashboardURL,
			OperationData: provisionResponse.OperationData,
			Metadata:      metadata,
		})
	} else {
		h.respond(w, http.StatusCreated, requestId, apiresponses.ProvisioningResponse{
			DashboardURL: provisionResponse.DashboardURL,
			Metadata:     metadata,
		})
	}
}
