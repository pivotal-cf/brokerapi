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

const getInstanceLogKey = "getInstance"

func (h APIHandler) GetInstance(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get instance endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(logutil.Join(getInstanceLogKey, middlewares.ApiVersionInvalidKey), logutil.Error(err))
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
			logger.Error(logutil.Join(getInstanceLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(getInstanceLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(getInstanceLogKey, unknownErrorKey), logutil.Error(err))
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
