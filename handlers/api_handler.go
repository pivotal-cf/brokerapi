package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/internal/blog"
)

const (
	invalidServiceDetailsErrorKey = "invalid-service-details"
	serviceIdMissingKey           = "service-id-missing"
	planIdMissingKey              = "plan-id-missing"
	unknownErrorKey               = "unknown-error"
)

var (
	serviceIdError        = errors.New("service_id missing")
	planIdError           = errors.New("plan_id missing")
	invalidServiceIDError = errors.New("service-id not in the catalog")
	invalidPlanIDError    = errors.New("plan-id not in the catalog")
)

type APIHandler struct {
	serviceBroker domain.ServiceBroker
	logger        blog.Blog
}

func NewApiHandler(broker domain.ServiceBroker, logger *slog.Logger) APIHandler {
	return APIHandler{serviceBroker: broker, logger: blog.New(logger)}
}

func (h APIHandler) respond(w http.ResponseWriter, status int, requestIdentity string, response any) {
	w.Header().Set("Content-Type", "application/json")
	if requestIdentity != "" {
		w.Header().Set("X-Broker-API-Request-Identity", requestIdentity)
	}
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(response)
	if err != nil {
		h.logger.Error("encoding response", err, slog.Int("status", status), slog.Any("response", response))
	}
}

type brokerVersion struct {
	Major int
	Minor int
}

func getAPIVersion(req *http.Request) brokerVersion {
	var version brokerVersion
	apiVersion := req.Header.Get("X-Broker-API-Version")

	fmt.Sscanf(apiVersion, "%d.%d", &version.Major, &version.Minor)

	return version
}
