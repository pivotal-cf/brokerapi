package handlers

import (
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

const unbindLogKey = "unbind"

func (h APIHandler) Unbind(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	bindingID := chi.URLParam(req, "binding_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID), slog.String(bindingIDLogKey, bindingID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	details := domain.UnbindDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
	}

	if details.ServiceID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		logger.Error(logutil.Join(unbindLogKey, serviceIdMissingKey), logutil.Error(serviceIdError))
		return
	}

	if details.PlanID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		logger.Error(logutil.Join(unbindLogKey, planIdMissingKey), logutil.Error(planIdError))
		return
	}

	asyncAllowed := req.FormValue("accepts_incomplete") == "true"
	unbindResponse, err := h.serviceBroker.Unbind(req.Context(), instanceID, bindingID, details, asyncAllowed)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(unbindLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(unbindLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(unbindLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if unbindResponse.IsAsync {
		h.respond(w, http.StatusAccepted, requestId, apiresponses.UnbindResponse{
			OperationData: unbindResponse.OperationData,
		})
	} else {
		h.respond(w, http.StatusOK, requestId, apiresponses.EmptyResponse{})
	}
}
