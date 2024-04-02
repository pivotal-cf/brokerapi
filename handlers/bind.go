package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/pivotal-cf/brokerapi/v11/internal/blog"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const (
	bindLogKey                 = "bind"
	invalidBindDetailsErrorKey = "invalid-bind-details"
)

func (h APIHandler) Bind(w http.ResponseWriter, req *http.Request) {
	instanceID := chi.URLParam(req, "instance_id")
	bindingID := chi.URLParam(req, "binding_id")

	logger := h.logger.Session(req.Context(), bindLogKey, blog.InstanceID(instanceID), blog.BindingID(bindingID))

	version := getAPIVersion(req)
	asyncAllowed := false
	if version.Minor >= 14 {
		asyncAllowed = req.FormValue("accepts_incomplete") == "true"
	}

	requestId := fmt.Sprintf("%v", req.Context().Value(middlewares.RequestIdentityKey))

	var details domain.BindDetails
	if err := json.NewDecoder(req.Body).Decode(&details); err != nil {
		logger.Error(invalidBindDetailsErrorKey, err)
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

	if details.PlanID == "" {
		logger.Error(planIdMissingKey, planIdError)
		h.respond(w, http.StatusBadRequest, requestId, apiresponses.ErrorResponse{
			Description: planIdError.Error(),
		})
		return
	}

	binding, err := h.serviceBroker.Bind(req.Context(), instanceID, bindingID, details, asyncAllowed)
	if err != nil {
		switch err := err.(type) {
		case *apiresponses.FailureResponse:
			statusCode := err.ValidatedStatusCode(slog.New(logger))
			errorResponse := err.ErrorResponse()
			if err == apiresponses.ErrInstanceDoesNotExist {
				// work around ErrInstanceDoesNotExist having different pre-refactor behaviour to other actions
				errorResponse = apiresponses.ErrorResponse{
					Description: err.Error(),
				}
				statusCode = http.StatusNotFound
			}
			logger.Error(err.LoggerAction(), err)
			h.respond(w, statusCode, requestId, errorResponse)
		default:
			logger.Error(unknownErrorKey, err)
			h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{
				Description: err.Error(),
			})
		}
		return
	}

	if binding.AlreadyExists {
		h.respond(w, http.StatusOK, requestId, apiresponses.BindingResponse{
			Credentials:     binding.Credentials,
			SyslogDrainURL:  binding.SyslogDrainURL,
			RouteServiceURL: binding.RouteServiceURL,
			VolumeMounts:    binding.VolumeMounts,
			BackupAgentURL:  binding.BackupAgentURL,
		})
		return
	}

	if binding.IsAsync {
		h.respond(w, http.StatusAccepted, requestId, apiresponses.AsyncBindResponse{
			OperationData: binding.OperationData,
		})
		return
	}

	if version.Minor == 8 || version.Minor == 9 {
		experimentalVols := []domain.ExperimentalVolumeMount{}

		for _, vol := range binding.VolumeMounts {
			experimentalConfig, err := json.Marshal(vol.Device.MountConfig)
			if err != nil {
				logger.Error(unknownErrorKey, err)
				h.respond(w, http.StatusInternalServerError, requestId, apiresponses.ErrorResponse{Description: err.Error()})
				return
			}

			experimentalVols = append(experimentalVols, domain.ExperimentalVolumeMount{
				ContainerPath: vol.ContainerDir,
				Mode:          vol.Mode,
				Private: domain.ExperimentalVolumeMountPrivate{
					Driver:  vol.Driver,
					GroupID: vol.Device.VolumeId,
					Config:  string(experimentalConfig),
				},
			})
		}

		experimentalBinding := apiresponses.ExperimentalVolumeMountBindingResponse{
			Credentials:     binding.Credentials,
			RouteServiceURL: binding.RouteServiceURL,
			SyslogDrainURL:  binding.SyslogDrainURL,
			VolumeMounts:    experimentalVols,
			BackupAgentURL:  binding.BackupAgentURL,
		}
		h.respond(w, http.StatusCreated, requestId, experimentalBinding)
		return
	}

	h.respond(w, http.StatusCreated, requestId, apiresponses.BindingResponse{
		Credentials:     binding.Credentials,
		SyslogDrainURL:  binding.SyslogDrainURL,
		RouteServiceURL: binding.RouteServiceURL,
		VolumeMounts:    binding.VolumeMounts,
		BackupAgentURL:  binding.BackupAgentURL,
	})
}
