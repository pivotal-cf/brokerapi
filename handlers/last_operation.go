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

const lastOperationLogKey = "lastOperation"

func (h APIHandler) LastOperation(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	pollDetails := domain.PollDetails{
		PlanID:        req.FormValue("plan_id"),
		ServiceID:     req.FormValue("service_id"),
		OperationData: req.FormValue("operation"),
	}

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	logger.Info(logutil.Join(lastOperationLogKey, "starting-check-for-operation"))

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	lastOperation, err := h.serviceBroker.LastOperation(req.Context(), instanceID, pollDetails)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(lastOperationLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(lastOperationLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(lastOperationLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	logger.Info(logutil.Join(lastOperationLogKey, "done-check-for-operation"), slog.Any("state", lastOperation.State))

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
		Description: lastOperation.Description,
	}

	h.respond(w, http.StatusOK, requestId, lastOperationResponse)
}
