package handlers

import (
	"fmt"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
	"github.com/pivotal-cf/brokerapi/v10/utils"
)

const getCatalogLogKey = "getCatalog"

func (h *APIHandler) Catalog(w http.ResponseWriter, req *http.Request) {
	logger := h.logger.Session(getCatalogLogKey, nil,
		utils.DataForContext(req.Context(), middlewares.CorrelationIDKey, middlewares.RequestIdentityKey))
	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	services, err := h.serviceBroker.Services(req.Context())
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

	catalog := apiresponses.CatalogResponse{
		Services: services,
	}

	h.respond(w, http.StatusOK, requestId, catalog)
}
