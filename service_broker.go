package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
)

type ServiceBroker interface {
	Services(context context.Context) []Service

	Provision(context context.Context, instanceID string, details ProvisionDetails, asyncAllowed bool) (ProvisionedServiceSpec, error)
	Deprovision(context context.Context, instanceID string, details DeprovisionDetails, asyncAllowed bool) (DeprovisionServiceSpec, error)

	Bind(context context.Context, instanceID, bindingID string, details BindDetails) (Binding, error)
	Unbind(context context.Context, instanceID, bindingID string, details UnbindDetails) error

	Update(context context.Context, instanceID string, details UpdateDetails, asyncAllowed bool) (UpdateServiceSpec, error)

	LastOperation(context context.Context, instanceID, operationData string) (LastOperation, error)
}

type ProvisionDetails struct {
	ServiceID        string          `json:"service_id"`
	PlanID           string          `json:"plan_id"`
	OrganizationGUID string          `json:"organization_guid"`
	SpaceGUID        string          `json:"space_guid"`
	RawParameters    json.RawMessage `json:"parameters,omitempty"`
}

type ProvisionedServiceSpec struct {
	IsAsync       bool
	DashboardURL  string
	OperationData string
}

type BindDetails struct {
	AppGUID       string          `json:"app_guid"`
	PlanID        string          `json:"plan_id"`
	ServiceID     string          `json:"service_id"`
	BindResource  *BindResource   `json:"bind_resource,omitempty"`
	RawParameters json.RawMessage `json:"parameters,omitempty"`
}

type BindResource struct {
	AppGuid string `json:"app_guid,omitempty"`
	Route   string `json:"route,omitempty"`
}

type UnbindDetails struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
}

type UpdateServiceSpec struct {
	IsAsync       bool
	OperationData string
}

type DeprovisionServiceSpec struct {
	IsAsync       bool
	OperationData string
}

type DeprovisionDetails struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
}

type UpdateDetails struct {
	ServiceID      string          `json:"service_id"`
	PlanID         string          `json:"plan_id"`
	RawParameters  json.RawMessage `json:"parameters,omitempty"`
	PreviousValues PreviousValues  `json:"previous_values"`
}

type PreviousValues struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
	OrgID     string `json:"organization_id"`
	SpaceID   string `json:"space_id"`
}

type LastOperation struct {
	State       LastOperationState
	Description string
}

type LastOperationState string

const (
	InProgress LastOperationState = "in progress"
	Succeeded  LastOperationState = "succeeded"
	Failed     LastOperationState = "failed"
)

type Binding struct {
	Credentials     interface{}   `json:"credentials"`
	SyslogDrainURL  string        `json:"syslog_drain_url,omitempty"`
	RouteServiceURL string        `json:"route_service_url,omitempty"`
	VolumeMounts    []VolumeMount `json:"volume_mounts,omitempty"`
}

type VolumeMount struct {
	Driver       string       `json:"driver"`
	ContainerDir string       `json:"container_dir"`
	Mode         string       `json:"mode"`
	DeviceType   string       `json:"device_type"`
	Device       SharedDevice `json:"device"`
}

type SharedDevice struct {
	VolumeId    string                 `json:"volume_id"`
	MountConfig map[string]interface{} `json:"mount_config"`
}

var (
	ErrInstanceAlreadyExists  = errors.New("instance already exists")
	ErrInstanceDoesNotExist   = errors.New("instance does not exist")
	ErrInstanceLimitMet       = errors.New("instance limit for this service has been reached")
	ErrPlanQuotaExceeded      = errors.New("The quota for this service plan has been exceeded. Please contact your Operator for help.")
	ErrBindingAlreadyExists   = errors.New("binding already exists")
	ErrBindingDoesNotExist    = errors.New("binding does not exist")
	ErrAsyncRequired          = errors.New("This service plan requires client support for asynchronous service operations.")
	ErrPlanChangeNotSupported = errors.New("The requested plan migration cannot be performed")
	ErrRawParamsInvalid       = errors.New("The format of the parameters is not valid JSON")
	ErrAppGuidNotProvided     = errors.New("app_guid is a required field but was not provided")
)
