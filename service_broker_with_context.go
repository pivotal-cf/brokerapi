package brokerapi

import "context"

type serviceBrokerWithContext struct {
	delegate ServiceBroker
}

func NewServiceBrokerWithContextFrom(serviceBroker ServiceBroker) ServiceBrokerWithContext {
	return &serviceBrokerWithContext{serviceBroker}
}

func (sbc *serviceBrokerWithContext) Services() []Service {
	return sbc.delegate.Services()
}

func (sbc *serviceBrokerWithContext) Provision(instanceID string, details ProvisionDetails, asyncAllowed bool, _ context.Context) (ProvisionedServiceSpec, error) {
	return sbc.delegate.Provision(instanceID, details, asyncAllowed)
}

func (sbc *serviceBrokerWithContext) Deprovision(instanceID string, details DeprovisionDetails, asyncAllowed bool, _ context.Context) (DeprovisionServiceSpec, error) {
	return sbc.delegate.Deprovision(instanceID, details, asyncAllowed)
}

func (sbc *serviceBrokerWithContext) Bind(instanceID, bindingID string, details BindDetails, _ context.Context) (Binding, error) {
	return sbc.delegate.Bind(instanceID, bindingID, details)
}

func (sbc *serviceBrokerWithContext) Unbind(instanceID, bindingID string, details UnbindDetails, _ context.Context) error {
	return sbc.delegate.Unbind(instanceID, bindingID, details)
}

func (sbc *serviceBrokerWithContext) Update(instanceID string, details UpdateDetails, asyncAllowed bool, _ context.Context) (UpdateServiceSpec, error) {
	return sbc.delegate.Update(instanceID, details, asyncAllowed)
}

func (sbc *serviceBrokerWithContext) LastOperation(instanceID, operationData string) (LastOperation, error) {
	return sbc.delegate.LastOperation(instanceID, operationData)
}
