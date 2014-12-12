package api

import "errors"

type ServiceBroker interface {
	Services() []Service

	Provision(instanceID string, serviceDetails ServiceDetails) error
	Deprovision(instanceID string) error

	Bind(instanceID, bindingID string) (interface{}, error)
	Unbind(instanceID, bindingID string) error
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
)
