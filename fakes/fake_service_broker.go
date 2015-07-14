package fakes

import "github.com/pivotal-cf/brokerapi"

type FakeServiceBroker struct {
	ProvisionDetails brokerapi.ProvisionDetails

	ProvisionedInstanceIDs   []string
	DeprovisionedInstanceIDs []string

	BoundInstanceIDs    []string
	BoundBindingIDs     []string
	BoundBindingDetails brokerapi.BindDetails

	InstanceLimit int

	ProvisionError   error
	BindError        error
	DeprovisionError error

	BrokerCalled bool
}

func (fakeBroker *FakeServiceBroker) Services() []brokerapi.Service {
	fakeBroker.BrokerCalled = true

	return []brokerapi.Service{
		brokerapi.Service{
			ID:          "0A789746-596F-4CEA-BFAC-A0795DA056E3",
			Name:        "p-cassandra",
			Description: "Cassandra service for application development and testing",
			Bindable:    true,
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

func (fakeBroker *FakeServiceBroker) Provision(instanceID string, details brokerapi.ProvisionDetails) error {
	fakeBroker.BrokerCalled = true

	if fakeBroker.ProvisionError != nil {
		return fakeBroker.ProvisionError
	}

	if len(fakeBroker.ProvisionedInstanceIDs) >= fakeBroker.InstanceLimit {
		return brokerapi.ErrInstanceLimitMet
	}

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return brokerapi.ErrInstanceAlreadyExists
	}

	fakeBroker.ProvisionDetails = details
	fakeBroker.ProvisionedInstanceIDs = append(fakeBroker.ProvisionedInstanceIDs, instanceID)
	return nil
}

func (fakeBroker *FakeServiceBroker) Deprovision(instanceID string) error {
	fakeBroker.BrokerCalled = true

	if fakeBroker.DeprovisionError != nil {
		return fakeBroker.DeprovisionError
	}

	fakeBroker.DeprovisionedInstanceIDs = append(fakeBroker.DeprovisionedInstanceIDs, instanceID)

	if sliceContains(instanceID, fakeBroker.ProvisionedInstanceIDs) {
		return nil
	}
	return brokerapi.ErrInstanceDoesNotExist
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
