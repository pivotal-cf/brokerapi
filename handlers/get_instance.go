package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v7/middlewares"
	"github.com/pivotal-cf/brokerapi/v7/utils"
)

const getInstanceLogKey = "getInstance"

func (h APIHandler) GetInstance(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	logger := h.logger.Session(getInstanceLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	}, utils.DataForContext(req.Context(), middlewares.CorrelationIDKey))

	ctx := req.Context()
	originatingIdentity := fmt.Sprintf("%v", ctx.Value("originatingIdentity"))

	version := getAPIVersion(req)
	if version.Minor < 14 {
		err := errors.New("get instance endpoint only supported starting with OSB version 2.14")
		h.respond(w, http.StatusPreconditionFailed, originatingIdentity, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		logger.Error(middlewares.ApiVersionInvalidKey, err)
		return
	}

	instanceDetails, err := h.serviceBroker.GetInstance(req.Context(), instanceID)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(logger), originatingIdentity, err.ErrorResponse())
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, originatingIdentity, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	h.respond(w, http.StatusOK, originatingIdentity, apiresponses.GetInstanceResponse{
		ServiceID:    instanceDetails.ServiceID,
		PlanID:       instanceDetails.PlanID,
		DashboardURL: instanceDetails.DashboardURL,
		Parameters:   instanceDetails.Parameters,
	})
}
