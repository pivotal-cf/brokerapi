package handlers

import (
	"errors"
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

const getBindLogKey = "getBinding"

func (h APIHandler) GetBinding(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	bindingID := chi.URLParam(req, "binding_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID), slog.String(bindingIDLogKey, bindingID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get binding endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(logutil.Join(getBindLogKey, middlewares.ApiVersionInvalidKey), logutil.Error(err))
		return
	}

	details := domain.FetchBindingDetails{
		ServiceID: req.URL.Query().Get("service_id"),
		PlanID:    req.URL.Query().Get("plan_id"),
	}

	binding, err := h.serviceBroker.GetBinding(req.Context(), instanceID, bindingID, details)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(getBindLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(getBindLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(getBindLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	h.respond(w, http.StatusOK, requestId, apiresponses.GetBindingResponse{
		BindingResponse: apiresponses.BindingResponse{
			Credentials:     binding.Credentials,
			SyslogDrainURL:  binding.SyslogDrainURL,
			RouteServiceURL: binding.RouteServiceURL,
			VolumeMounts:    binding.VolumeMounts,
		},
		Parameters: binding.Parameters,
	})
}
