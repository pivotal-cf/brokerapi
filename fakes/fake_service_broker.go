package fakes

import "github.com/pivotal-cf/brokerapi"

type FakeServiceBroker struct {
	ProvisionDetails brokerapi.ProvisionDetails
	UpdateDetails    brokerapi.UpdateDetails

	ProvisionedInstanceIDs   []string
	DeprovisionedInstanceIDs []string
	UpdatedInstanceIDs       []string

	BoundInstanceIDs    []string
	BoundBindingIDs     []string
	BoundBindingDetails brokerapi.BindDetails

	InstanceLimit int

	ProvisionError     error
	BindError          error
	DeprovisionError   error
	LastOperationError error
	UpdateError        error

	BrokerCalled             bool
	LastOperationState       brokerapi.LastOperationState
	LastOperationDescription string

	ShouldReturnAsync brokerapi.IsAsync
}

type FakeAsyncServiceBroker struct {
	FakeServiceBroker
	ShouldProvisionAsync bool
}

type FakeAsyncOnlyServiceBroker struct {
	FakeServiceBroker
}

func (fakeBroker *FakeServiceBroker) Services() []brokerapi.Service {
	fakeBroker.BrokerCalled = true

	return []brokerapi.Service{
		brokerapi.Service{
			ID:            "0A789746-596F-4CEA-BFAC-A0795DA056E3",
			Name:          "p-cassandra",
			Description:   "Cassandra service for application development and testing",
			Bindable:      true,
			PlanUpdatable: true,
			Plans: []brokerapi.ServicePlan{
				brokerapi.ServicePlan{
					ID:          "ABE176EE-F69F-4A96-80CE-142595CC24E3",
					Name:        "default",
					Description: "The default Cassandra plan",
					Metadata: brokerapi.ServicePlanMetadata{
						Bullets:     []string{},
						DisplayName: "Cassandra",
					},
				},
			},
			Metadata: brokerapi.ServiceMetadata{
				DisplayName:      "Cassandra",
				LongDescription:  "Long description",
				DocumentationUrl: "http://thedocs.com",
				SupportUrl:       "http://helpme.no",
				Listing: brokerapi.ServiceMetadataListing{
					Blurb:    "blah blah",
					ImageUrl: "http://foo.com/thing.png",
				},
				Provider: brokerapi.ServiceMetadataProvider{
					Name: "Pivotal",
				},
			},
			Tags: []string{
				"pivotal",
				"cassandra",
			},
		},
	}
}

func (fakeBroker *FakeServiceBroker) Provision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.ProvisionError != nil {
		return false, fakeBroker.ProvisionError
	}

	if len(fakeBroker.ProvisionedInstanceIDs) >= fakeBroker.InstanceLimit {
		return false, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return false, brokerapi.ErrInstanceAlreadyExists
	}

	fakeBroker.ProvisionDetails = details
	fakeBroker.ProvisionedInstanceIDs = append(fakeBroker.ProvisionedInstanceIDs, instanceID)
	return false, nil
}

func (fakeBroker *FakeAsyncServiceBroker) Provision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.ProvisionError != nil {
		return false, fakeBroker.ProvisionError
	}

	if len(fakeBroker.ProvisionedInstanceIDs) >= fakeBroker.InstanceLimit {
		return false, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return false, brokerapi.ErrInstanceAlreadyExists
	}

	fakeBroker.ProvisionDetails = details
	fakeBroker.ProvisionedInstanceIDs = append(fakeBroker.ProvisionedInstanceIDs, instanceID)
	return brokerapi.IsAsync(fakeBroker.ShouldProvisionAsync), nil
}

func (fakeBroker *FakeAsyncOnlyServiceBroker) Provision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.ProvisionError != nil {
		return false, fakeBroker.ProvisionError
	}

	if len(fakeBroker.ProvisionedInstanceIDs) >= fakeBroker.InstanceLimit {
		return false, brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return false, brokerapi.ErrInstanceAlreadyExists
	}

	if !asyncAllowed {
		return true, brokerapi.ErrAsyncRequired
	}

	fakeBroker.ProvisionDetails = details
	fakeBroker.ProvisionedInstanceIDs = append(fakeBroker.ProvisionedInstanceIDs, instanceID)
	return true, nil
}

func (fakeBroker *FakeServiceBroker) Update(instanceID string, details brokerapi.UpdateDetails) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.UpdateError != nil {
		return false, fakeBroker.UpdateError
	}

	fakeBroker.UpdateDetails = details
	fakeBroker.UpdatedInstanceIDs = append(fakeBroker.UpdatedInstanceIDs, instanceID)
	return fakeBroker.ShouldReturnAsync, nil
}

func (fakeBroker *FakeServiceBroker) Deprovision(instanceID string, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.DeprovisionError != nil {
		return brokerapi.IsAsync(false), fakeBroker.DeprovisionError
	}

	fakeBroker.DeprovisionedInstanceIDs = append(fakeBroker.DeprovisionedInstanceIDs, instanceID)

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return brokerapi.IsAsync(false), nil
	}
	return brokerapi.IsAsync(false), brokerapi.ErrInstanceDoesNotExist
}

func (fakeBroker *FakeAsyncOnlyServiceBroker) Deprovision(instanceID string, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.DeprovisionError != nil {
		return brokerapi.IsAsync(true), fakeBroker.DeprovisionError
	}

	if !asyncAllowed {
		return brokerapi.IsAsync(true), brokerapi.ErrAsyncRequired
	}

	fakeBroker.DeprovisionedInstanceIDs = append(fakeBroker.DeprovisionedInstanceIDs, instanceID)

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return brokerapi.IsAsync(true), nil
	}

	return brokerapi.IsAsync(true), brokerapi.ErrInstanceDoesNotExist
}

func (fakeBroker *FakeAsyncServiceBroker) Deprovision(instanceID string, asyncAllowed bool) (brokerapi.IsAsync, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.DeprovisionError != nil {
		return brokerapi.IsAsync(asyncAllowed), fakeBroker.DeprovisionError
	}

	fakeBroker.DeprovisionedInstanceIDs = append(fakeBroker.DeprovisionedInstanceIDs, instanceID)

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return brokerapi.IsAsync(asyncAllowed), nil
	}

	return brokerapi.IsAsync(asyncAllowed), brokerapi.ErrInstanceDoesNotExist
}

func (fakeBroker *FakeServiceBroker) Bind(instanceID, bindingID string, details brokerapi.BindDetails) (interface{}, error) {
	fakeBroker.BrokerCalled = true

	if fakeBroker.BindError != nil {
		return nil, fakeBroker.BindError
	}

	fakeBroker.BoundBindingDetails = details

	fakeBroker.BoundInstanceIDs = append(fakeBroker.BoundInstanceIDs, instanceID)
	fakeBroker.BoundBindingIDs = append(fakeBroker.BoundBindingIDs, bindingID)

	return FakeCredentials{
		Host:     "127.0.0.1",
		Port:     3000,
		Username: "batman",
		Password: "robin",
	}, nil
}

func (fakeBroker *FakeServiceBroker) Unbind(instanceID, bindingID string) error {
	fakeBroker.BrokerCalled = true

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		if sliceContains(bindingID, fakeBroker.BoundBindingIDs) {
			return nil
		}
		return brokerapi.ErrBindingDoesNotExist
	}

	return brokerapi.ErrInstanceDoesNotExist
}

func (fakeBroker *FakeServiceBroker) LastOperation(instanceID string) (brokerapi.LastOperation, error) {

	if fakeBroker.LastOperationError != nil {
		return brokerapi.LastOperation{}, fakeBroker.LastOperationError
	}

	return brokerapi.LastOperation{State: fakeBroker.LastOperationState, Description: fakeBroker.LastOperationDescription}, nil
}

type FakeCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func sliceContains(needle string, haystack []string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}
