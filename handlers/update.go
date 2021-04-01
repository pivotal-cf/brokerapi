package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v8/middlewares"
	"github.com/pivotal-cf/brokerapi/v8/utils"
)

const updateLogKey = "update"

func (h APIHandler) Update(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["instance_id"]

	logger := h.logger.Session(updateLogKey, lager.Data{
		instanceIDLogKey: instanceID,
	}, utils.DataForContext(req.Context(), middlewares.CorrelationIDKey))

	requestId := fmt.Sprintf("%v", req.Context().Value("requestIdentity"))

	var details domain.UpdateDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		h.logger.Error(invalidServiceDetailsErrorKey, err)
		h.respond(w, http.StatusUnprocessableEntity, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	if details.ServiceID == "" {
		logger.Error(serviceIdMissingKey, serviceIdError)
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
			h.logger.Error(err.LoggerAction(), err)
			h.respond(w, err.ValidatedStatusCode(h.logger), requestId, err.ErrorResponse())
		default:
			h.logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	statusCode := http.StatusOK
	if updateServiceSpec.IsAsync {
		statusCode = http.StatusAccepted
	}
	h.respond(w, statusCode, requestId, apiresponses.UpdateResponse{
		OperationData: updateServiceSpec.OperationData,
		DashboardURL:  updateServiceSpec.DashboardURL,
	})
}
