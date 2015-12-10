package brokerapi

import "errors"

type ServiceBroker interface {
	Services() []Service

	Provision(instanceID string, details ProvisionDetails, asyncAllowed bool) (IsAsync, error)

	Deprovision(instanceID string, asyncAllowed bool) (IsAsync, error)

	Bind(instanceID, bindingID string, details BindDetails) (interface{}, error)
	Unbind(instanceID, bindingID string) error

	LastOperation(instanceID string) (LastOperation, error)
}

type IsAsync bool

type ProvisionDetails struct {
	ID               string                 `json:"service_id"`
	PlanID           string                 `json:"plan_id"`
	OrganizationGUID string                 `json:"organization_guid"`
	SpaceGUID        string                 `json:"space_guid"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

type BindDetails struct {
	AppGUID    string                 `json:"app_guid"`
	PlanID     string                 `json:"plan_id"`
	ServiceID  string                 `json:"service_id"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
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

var (
	ErrInstanceAlreadyExists = errors.New("instance already exists")
	ErrInstanceDoesNotExist  = errors.New("instance does not exist")
	ErrInstanceLimitMet      = errors.New("instance limit for this service has been reached")
	ErrBindingAlreadyExists  = errors.New("binding already exists")
	ErrBindingDoesNotExist   = errors.New("binding does not exist")
	ErrAsyncRequired         = errors.New("This service plan requires client support for asynchronous service operations.")
)
