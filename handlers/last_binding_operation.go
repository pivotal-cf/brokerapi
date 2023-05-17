package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v9/domain"
	"github.com/pivotal-cf/brokerapi/v9/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v9/middlewares"
	"github.com/pivotal-cf/brokerapi/v9/utils"
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

	logger := h.logger.Session(lastBindingOperationLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	}, utils.DataForContext(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey))

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
			h.respond(w, err.ValidatedStatusCode(logger), requestId, err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	logger.WithData(lager.Data{"state": lastOperation.State}).Info("done-check-for-binding-operation")

	lastOperationResponse := apiresponses.LastOperationResponse{
		State:       lastOperation.State,
		Description: lastOperation.Description,
	}
	h.respond(w, http.StatusOK, requestId, lastOperationResponse)
}
