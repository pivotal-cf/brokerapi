package api

import "errors"

type ServiceBroker interface {
	Services() []Service

	Provision(instanceID string, params map[string]string) error
	Deprovision(instanceID string) error

	Bind(instanceID, bindingID string) (interface{}, error)
	Unbind(instanceID, bindingID string) error
}

var (
	ErrInstanceAlreadyExists = errors.New("instance already exists")
	ErrInstanceDoesNotExist  = errors.New("instance does not exist")
	ErrInstanceLimitMet      = errors.New("instance limit for this service has been reached")
	ErrBindingAlreadyExists  = errors.New("binding already exists")
	ErrBindingDoesNotExist   = errors.New("binding does not exist")
)
