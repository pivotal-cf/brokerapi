package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pivotal-cf/brokerapi/v10/internal/logutil"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
	"github.com/pivotal-cf/brokerapi/v10/utils"
)

const updateLogKey = "update"

func (h APIHandler) Update(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.With(append(
		[]any{slog.String(instanceIDLogKey, instanceID)},
		utils.ContextAttr(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey)...,
	)...)

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	var details domain.UpdateDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(logutil.Join(updateLogKey, invalidServiceDetailsErrorKey), logutil.Error(err))
		h.respond(w, http.StatusUnprocessableEntity, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	if details.ServiceID == "" {
		logger.Error(logutil.Join(updateLogKey, serviceIdMissingKey), logutil.Error(serviceIdError))
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		return
	}

	acceptsIncompleteFlag, _ := strconv.ParseBool(req.URL.Query().Get("accepts_incomplete"))

	updateServiceSpec, err := h.serviceBroker.Update(req.Context(), instanceID, details, acceptsIncompleteFlag)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(logutil.Join(updateLogKey, err.LoggerAction()), logutil.Error(err))
			h.respond(w, err.ValidatedStatusCode(updateLogKey, logger), requestId, err.ErrorResponse())
		default:
			logger.Error(logutil.Join(updateLogKey, unknownErrorKey), logutil.Error(err))
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	var metadata interface{}
	if !updateServiceSpec.Metadata.IsEmpty() {
		metadata = updateServiceSpec.Metadata
	}

	statusCode := http.StatusOK
	if updateServiceSpec.IsAsync {
		statusCode = http.StatusAccepted
	}
	h.respond(w, statusCode, requestId, apiresponses.UpdateResponse{
		OperationData: updateServiceSpec.OperationData,
		DashboardURL:  updateServiceSpec.DashboardURL,
		Metadata:      metadata,
	})
}
