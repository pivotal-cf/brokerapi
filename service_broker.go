package brokerapi

import "errors"

type ServiceBroker interface {
	Services() []Service

	Provision(instanceID string, details ProvisionDetails, asyncAllowed bool) (ProvisionedServiceSpec, error)
	Deprovision(instanceID string, details DeprovisionDetails, asyncAllowed bool) (IsAsync, error)

	Bind(instanceID, bindingID string, details BindDetails) (Binding, error)
	Unbind(instanceID, bindingID string, details UnbindDetails) error

	Update(instanceID string, details UpdateDetails, asyncAllowed bool) (IsAsync, error)

	LastOperation(instanceID string) (LastOperation, error)
}

type IsAsync bool

type ProvisionDetails struct {
	ServiceID        string                 `json:"service_id"`
	PlanID           string                 `json:"plan_id"`
	OrganizationGUID string                 `json:"organization_guid"`
	SpaceGUID        string                 `json:"space_guid"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

type ProvisionedServiceSpec struct {
	IsAsync      bool
	DashboardURL string
}

type BindDetails struct {
	AppGUID    string                 `json:"app_guid"`
	PlanID     string                 `json:"plan_id"`
	ServiceID  string                 `json:"service_id"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type UnbindDetails struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
}

type DeprovisionDetails struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
}

type UpdateDetails struct {
	ServiceID      string                 `json:"service_id"`
	PlanID         string                 `json:"plan_id"`
	Parameters     map[string]interface{} `json:"parameters"`
	PreviousValues PreviousValues         `json:"previous_values"`
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
	Credentials    interface{} `json:"credentials"`
	SyslogDrainURL string      `json:"syslog_drain_url,omitempty"`
}

var (
	ErrInstanceAlreadyExists  = errors.New("instance already exists")
	ErrInstanceDoesNotExist   = errors.New("instance does not exist")
	ErrInstanceLimitMet       = errors.New("instance limit for this service has been reached")
	ErrBindingAlreadyExists   = errors.New("binding already exists")
	ErrBindingDoesNotExist    = errors.New("binding does not exist")
	ErrAsyncRequired          = errors.New("This service plan requires client support for asynchronous service operations")
	ErrPlanChangeNotSupported = errors.New("The requested plan migration cannot be performed")
)
