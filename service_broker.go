package brokerapi

import "errors"

type ServiceBroker interface {
	Services() []Service

	ProvisionSync(instanceID string, serviceDetails ServiceDetails) (error)
	ProvisionAsync(instanceID string, serviceDetails ServiceDetails) (error)

	Deprovision(instanceID string) error

	Bind(instanceID, bindingID string) (interface{}, error)
	Unbind(instanceID, bindingID string) error

	LastOperation(instanceID string) (*LastOperation, error)
}

type ProvisionAsync bool

type LastOperation struct {
	State       string
	Description string
}

type ServiceDetails struct {
	ID               string `json:"service_id"`
	PlanID           string `json:"plan_id"`
	OrganizationGUID string `json:"organization_guid"`
	SpaceGUID        string `json:"space_guid"`
}

var (
	ErrInstanceAlreadyExists = errors.New("instance already exists")
	ErrInstanceDoesNotExist  = errors.New("instance does not exist")
	ErrInstanceLimitMet      = errors.New("instance limit for this service has been reached")
	ErrBindingAlreadyExists  = errors.New("binding already exists")
	ErrBindingDoesNotExist   = errors.New("binding does not exist")
	ErrInvalidAsyncProvision = errors.New("broker attempted to provision asynchronously when not supported by the caller")
	ErrAsyncRequired         = errors.New("This service plan requires client support for asynchronous service operations.")
)
