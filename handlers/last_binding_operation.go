package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v11/internal/blog"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const lastBindingOperationLogKey = "lastBindingOperation"

func (h APIHandler) LastBindingOperation(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	bindingID := chi.URLParam(req, "binding_id")
	pollDetails := domain.PollDetails{
		PlanID:        req.FormValue("plan_id"),
		ServiceID:     req.FormValue("service_id"),
		OperationData: req.FormValue("operation"),
	}

	logger := h.logger.Session(req.Context(), lastBindingOperationLogKey, blog.InstanceID(instanceID), blog.BindingID(bindingID))

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get binding endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(middlewares.ApiVersionInvalidKey, err)
		return
	}

	logger.Info("starting-check-for-binding-operation")

	lastOperation, err := h.serviceBroker.LastBindingOperation(req.Context(), instanceID, bindingID, pollDetails)
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

	logger.Info("done-check-for-binding-operation", slog.Any("state", lastOperation.State))

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
		Description: lastOperation.Description,
	}
	h.respond(w, http.StatusOK, requestId, lastOperationResponse)
}
