package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v11/internal/blog"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const lastOperationLogKey = "lastOperation"

func (h APIHandler) LastOperation(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	pollDetails := domain.PollDetails{
		PlanID:        req.FormValue("plan_id"),
		ServiceID:     req.FormValue("service_id"),
		OperationData: req.FormValue("operation"),
	}

	logger := h.logger.Session(req.Context(), lastOperationLogKey, blog.InstanceID(instanceID))

	logger.Info("starting-check-for-operation")

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	lastOperation, err := h.serviceBroker.LastOperation(req.Context(), instanceID, pollDetails)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(slog.New(logger)), requestId, err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	logger.Info("done-check-for-operation", slog.Any("state", lastOperation.State))

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
		Description: lastOperation.Description,
	}

	h.respond(w, http.StatusOK, requestId, lastOperationResponse)
}
