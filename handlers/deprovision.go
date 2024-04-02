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

const deprovisionLogKey = "deprovision"

func (h APIHandler) Deprovision(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")

	logger := h.logger.Session(req.Context(), deprovisionLogKey, blog.InstanceID(instanceID))

	details := domain.DeprovisionDetails{
		PlanID:    req.FormValue("plan_id"),
		ServiceID: req.FormValue("service_id"),
		Force:     req.FormValue("force") == "true",
	}

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	if details.ServiceID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: serviceIdError.Error(),
		})
		logger.Error(serviceIdMissingKey, serviceIdError)
		return
	}

	if details.PlanID == "" {
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		logger.Error(planIdMissingKey, planIdError)
		return
	}

	asyncAllowed := req.FormValue("accepts_incomplete") == "true"

	deprovisionSpec, err := h.serviceBroker.Deprovision(req.Context(), instanceID, details, asyncAllowed)
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

	if deprovisionSpec.IsAsync {
		h.respond(w, http.StatusAccepted, requestId, apiresponses.DeprovisionResponse{OperationData: deprovisionSpec.OperationData})
	} else {
		h.respond(w, http.StatusOK, requestId, apiresponses.EmptyResponse{})
	}
}
