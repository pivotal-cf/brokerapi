# brokerapi

[![Build Status](https://travis-ci.org/pivotal-cf/brokerapi.svg?branch=master)](https://travis-ci.org/pivotal-cf/brokerapi)

A go package for building V2 CF Service Brokers in Go. Depends on
[lager](https://github.com/pivotal-golang/lager) and
[gorilla/mux](https://github.com/gorilla/mux).

Requires go 1.4 or greater.

## Usage

`brokerapi` defines a `ServiceBroker` interface with 5 methods. Simply create
a concrete type that implements these methods, and pass an instance of it to
`brokerapi.New`, along with a `lager.Logger` for logging and a
`brokerapi.BrokerCredentials` containing some HTTP basic auth credentials.

e.g.

```
package main

import (
    "github.com/pivotal-cf/brokerapi"
    "github.com/pivotal-golang/lager"
)

type myServiceBroker struct {}

func (*myServiceBroker) Services() []brokerapi.Service {
    // Return a []brokerapi.Service here, describing your service(s) and plan(s)
}

func (*myServiceBroker) Provision(instanceID string, details brokerapi.ProvisionDetails) error {
    // Provision a new instance here
}

func (*myServiceBroker) Deprovision(instanceID string) error {
    // Deprovision instances here
}

func (*myServiceBroker) Bind(instanceID, bindingID string, details brokerapi.BindDetails) (interface{}, error) {
    // Bind to instances here
    // Return credentials which will be marshalled to JSON
}

func (*myServiceBroker) Unbind(instanceID, bindingID string) error {
    // Unbind from instances here
}

func main() {
    serviceBroker := &myServiceBroker{}
    logger := lager.NewLogger("my-service-broker")
    credentials := brokerapi.BrokerCredentials{
        Username: "username",
        Password: "password",
    }

    brokerAPI := brokerapi.New(serviceBroker, logger, credentials)
    http.Handle("/", brokerAPI)
    http.ListenAndServe(":3000", nil)
}
```

### Errors

`brokerapi` defines a handful of error types in `service_broker.go` for some
common error cases that your service broker may encounter. Return these from
your `ServiceBroker` methods where appropriate, and `brokerapi` will do the
right thing, and give Cloud Foundry an appropriate status code, as per the V2
Service Broker API specification.

The error types are:

```
ErrInstanceAlreadyExists
ErrInstanceDoesNotExist
ErrInstanceLimitMet
ErrBindingAlreadyExists
ErrBindingDoesNotExist
```

## Change Notes

* [d97ebdd](https://github.com/pivotal-cf/brokerapi/commit/d97ebddb70b3f099ec931e23a37bc70e82efb827) adds a new map property to the `brokerapi.BindDetails` struct in order to support arbitrary bind parameters. This allows API clients to send configuration parameters with their bind request.
* Starting with [10997ba](https://github.com/pivotal-cf/brokerapi/commit/10997baae7e5a4f1bc8db90afe402d509744ec48) the `Bind` function now takes an additional input parameter of type `brokerapi.BindDetails`. The corresponding struct specifies bind-specific properties sent by the CF API client.
* [8d9dd34](https://github.com/pivotal-cf/brokerapi/commit/8d9dd345ddd00d70c9aeaafb06ad3bed2213e0ea) adds support for arbitrary provision parameters. The broker can access the `Parameters` map in `brokerapi.ProvisionDetails` to lookup any configuration parameters sent by the client as part of their provision request.
