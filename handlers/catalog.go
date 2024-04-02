package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const getCatalogLogKey = "getCatalog"

func (h *APIHandler) Catalog(w http.ResponseWriter, req *http.Request) {
	logger := h.logger.Session(req.Context(), getCatalogLogKey)
	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	services, err := h.serviceBroker.Services(req.Context())
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

	catalog := apiresponses.CatalogResponse{
		Services: services,
	}

	h.respond(w, http.StatusOK, requestId, catalog)
}
