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

const getBindLogKey = "getBinding"

func (h APIHandler) GetBinding(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	bindingID := chi.URLParam(req, "binding_id")

	logger := h.logger.Session(req.Context(), getBindLogKey, blog.InstanceID(instanceID), blog.BindingID(bindingID))

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

	details := domain.FetchBindingDetails{
		ServiceID: req.URL.Query().Get("service_id"),
		PlanID:    req.URL.Query().Get("plan_id"),
	}

	binding, err := h.serviceBroker.GetBinding(req.Context(), instanceID, bindingID, details)
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
