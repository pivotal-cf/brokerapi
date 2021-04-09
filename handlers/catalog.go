package handlers

import (
	"fmt"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v8/middlewares"

	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
)

func (h *APIHandler) Catalog(w http.ResponseWriter, req *http.Request) {
	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	services, err := h.serviceBroker.Services(req.Context())
	if err != nil {
		h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	catalog := apiresponses.CatalogResponse{
		Services: services,
	}

	h.respond(w, http.StatusOK, requestId, catalog)
}
