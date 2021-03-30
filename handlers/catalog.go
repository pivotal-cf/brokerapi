package handlers

import (
	"fmt"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
)

func (h *APIHandler) Catalog(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	originatingIdentity := fmt.Sprintf("%v", ctx.Value("originatingIdentity"))

	services, err := h.serviceBroker.Services(req.Context())
	if err != nil {
		h.respond(w, http.StatusInternalServerError, originatingIdentity, apiresponses.ErrorResponse{
			Description: err.Error(),
		})
		return
	}

	catalog := apiresponses.CatalogResponse{
		Services: services,
	}

	h.respond(w, http.StatusOK, originatingIdentity, catalog)
}
