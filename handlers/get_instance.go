package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v11/internal/blog"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const getInstanceLogKey = "getInstance"

func (h APIHandler) GetInstance(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.Session(req.Context(), getInstanceLogKey, blog.InstanceID(instanceID))

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get instance endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(middlewares.ApiVersionInvalidKey, err)
		return
	}

	details := domain.FetchInstanceDetails{
		ServiceID: req.URL.Query().Get("service_id"),
		PlanID:    req.URL.Query().Get("plan_id"),
	}

	instanceDetails, err := h.serviceBroker.GetInstance(req.Context(), instanceID, details)
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

	var metadata interface{}
	if !instanceDetails.Metadata.IsEmpty() {
		metadata = instanceDetails.Metadata
	}

	h.respond(w, http.StatusOK, requestId, apiresponses.GetInstanceResponse{
		ServiceID:    instanceDetails.ServiceID,
		PlanID:       instanceDetails.PlanID,
		DashboardURL: instanceDetails.DashboardURL,
		Parameters:   instanceDetails.Parameters,
		Metadata:     metadata,
	})
}
