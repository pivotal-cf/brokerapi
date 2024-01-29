package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v10/internal/logutil"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
	"github.com/pivotal-cf/brokerapi/v10/utils"
)

const deprovisionLogKey = "deprovision"

func (h APIHandler) Deprovision(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	details := domain.DeprovisionDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
		Force:     req.FormValue("force") == "true",
	}

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	if details.ServiceID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		logger.Error(logutil.Join(deprovisionLogKey, serviceIdMissingKey), logutil.Error(serviceIdError))
		return
	}

	if details.PlanID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		logger.Error(logutil.Join(deprovisionLogKey, planIdMissingKey), logutil.Error(planIdError))
		return
	}

	asyncAllowed := req.FormValue("accepts_incomplete") == "true"

	deprovisionSpec, err := h.serviceBroker.Deprovision(req.Context(), instanceID, details, asyncAllowed)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(deprovisionLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(deprovisionLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(deprovisionLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if deprovisionSpec.IsAsync {
		h.respond(w, http.StatusAccepted, requestId, apiresponses.DeprovisionResponse{OperationData: deprovisionSpec.OperationData})
	} else {
		h.respond(w, http.StatusOK, requestId, apiresponses.EmptyResponse{})
	}
}
