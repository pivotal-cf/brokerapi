// Copyright (C) 2015-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

//Each method of the ServiceBroker interface maps to an individual endpoint of the Open Service Broker API.
//
//The specification is available here: https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md
//
//The OpenAPI documentation is available here: http://petstore.swagger.io/?url=https://raw.githubusercontent.com/openservicebrokerapi/servicebroker/v2.14/openapi.yaml
type ServiceBroker interface {
	//Catalog
	//
	//get the catalog of services that the service broker offers
	//  GET /v2/catalog
	Services(ctx context.Context) ([]Service, error)

	//ServiceInstances
	//
	//provision a service instance
	//  PUT /v2/service_instances/{instance_id}
	Provision(ctx context.Context, instanceID string, details ProvisionDetails, asyncAllowed bool) (ProvisionedServiceSpec, error)

	//deprovision a service instance
	//  DELETE /v2/service_instances/{instance_id}
	Deprovision(ctx context.Context, instanceID string, details DeprovisionDetails, asyncAllowed bool) (DeprovisionServiceSpec, error)

	//gets a service instnace
	//  GET /v2/service_instances/{instance_id}
	GetInstance(ctx context.Context, instanceID string) (GetInstanceDetailsSpec, error)

	//updates a service instance
	//  PATCH /v2/service_instances/{instance_id}
	Update(ctx context.Context, instanceID string, details UpdateDetails, asyncAllowed bool) (UpdateServiceSpec, error)

	//last requested operation state for service instance
	//  GET /v2/service_instances/{instance_id}/last_operation
	LastOperation(ctx context.Context, instanceID string, details PollDetails) (LastOperation, error)

	//ServiceBindings
	//
	//generation of a service binding
	//`PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}`
	Bind(ctx context.Context, instanceID, bindingID string, details BindDetails, asyncAllowed bool) (Binding, error)

	//deprovision of a service binding
	//`DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}`
	Unbind(ctx context.Context, instanceID, bindingID string, details UnbindDetails, asyncAllowed bool) (UnbindSpec, error)

	//gets a service binding
	//`GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}`
	GetBinding(ctx context.Context, instanceID, bindingID string) (GetBindingSpec, error)

	//last requested operation state for service binding
	//`GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation`
	LastBindingOperation(ctx context.Context, instanceID, bindingID string, details PollDetails) (LastOperation, error)
}

type DetailsWithRawParameters interface {
	GetRawParameters() json.RawMessage
}

type DetailsWithRawContext interface {
	GetRawContext() json.RawMessage
}

func (d ProvisionDetails) GetRawContext() json.RawMessage {
	return d.RawContext
}

func (d ProvisionDetails) GetRawParameters() json.RawMessage {
	return d.RawParameters
}

func (d BindDetails) GetRawContext() json.RawMessage {
	return d.RawContext
}

func (d BindDetails) GetRawParameters() json.RawMessage {
	return d.RawParameters
}

func (d UpdateDetails) GetRawParameters() json.RawMessage {
	return d.RawParameters
}

type ProvisionDetails struct {
	ServiceID        string          `json:"service_id"`
	PlanID           string          `json:"plan_id"`
	OrganizationGUID string          `json:"organization_guid"`
	SpaceGUID        string          `json:"space_guid"`
	RawContext       json.RawMessage `json:"context,omitempty"`
	RawParameters    json.RawMessage `json:"parameters,omitempty"`
}

type ProvisionedServiceSpec struct {
	IsAsync       bool
	DashboardURL  string
	OperationData string
}

type GetInstanceDetailsSpec struct {
	ServiceID    string      `json:"service_id"`
	PlanID       string      `json:"plan_id"`
	DashboardURL string      `json:"dashboard_url"`
	Parameters   interface{} `json:"parameters"`
}

type UnbindSpec struct {
	IsAsync       bool
	OperationData string
}

type BindDetails struct {
	AppGUID       string          `json:"app_guid"`
	PlanID        string          `json:"plan_id"`
	ServiceID     string          `json:"service_id"`
	BindResource  *BindResource   `json:"bind_resource,omitempty"`
	RawContext    json.RawMessage `json:"context,omitempty"`
	RawParameters json.RawMessage `json:"parameters,omitempty"`
}

type BindResource struct {
	AppGuid            string `json:"app_guid,omitempty"`
	SpaceGuid          string `json:"space_guid,omitempty"`
	Route              string `json:"route,omitempty"`
	CredentialClientID string `json:"credential_client_id,omitempty"`
}

type UnbindDetails struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
}

type UpdateServiceSpec struct {
	IsAsync       bool
	DashboardURL  string
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
	RawContext     json.RawMessage `json:"context,omitempty"`
}

type PreviousValues struct {
	PlanID    string `json:"plan_id"`
	ServiceID string `json:"service_id"`
	OrgID     string `json:"organization_id"`
	SpaceID   string `json:"space_id"`
}

type PollDetails struct {
	ServiceID     string `json:"service_id"`
	PlanID        string `json:"plan_id"`
	OperationData string `json:"operation"`
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
	IsAsync         bool          `json:"is_async"`
	OperationData   string        `json:"operation_data"`
	Credentials     interface{}   `json:"credentials"`
	SyslogDrainURL  string        `json:"syslog_drain_url"`
	RouteServiceURL string        `json:"route_service_url"`
	VolumeMounts    []VolumeMount `json:"volume_mounts"`
}

type GetBindingSpec struct {
	Credentials     interface{}
	SyslogDrainURL  string
	RouteServiceURL string
	VolumeMounts    []VolumeMount
	Parameters      interface{}
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

const (
	instanceExistsMsg           = "instance already exists"
	instanceDoesntExistMsg      = "instance does not exist"
	serviceLimitReachedMsg      = "instance limit for this service has been reached"
	servicePlanQuotaExceededMsg = "The quota for this service plan has been exceeded. Please contact your Operator for help."
	serviceQuotaExceededMsg     = "The quota for this service has been exceeded. Please contact your Operator for help."
	bindingExistsMsg            = "binding already exists"
	bindingDoesntExistMsg       = "binding does not exist"
	bindingNotFoundMsg          = "binding cannot be fetched"
	asyncRequiredMsg            = "This service plan requires client support for asynchronous service operations."
	planChangeUnsupportedMsg    = "The requested plan migration cannot be performed"
	rawInvalidParamsMsg         = "The format of the parameters is not valid JSON"
	appGuidMissingMsg           = "app_guid is a required field but was not provided"
	concurrentInstanceAccessMsg = "instance is being updated and cannot be retrieved"
)

var (
	ErrInstanceAlreadyExists = NewFailureResponseBuilder(
		errors.New(instanceExistsMsg), http.StatusConflict, instanceAlreadyExistsErrorKey,
	).WithEmptyResponse().Build()

	ErrInstanceDoesNotExist = NewFailureResponseBuilder(
		errors.New(instanceDoesntExistMsg), http.StatusGone, instanceMissingErrorKey,
	).WithEmptyResponse().Build()

	ErrInstanceLimitMet = NewFailureResponse(
		errors.New(serviceLimitReachedMsg), http.StatusInternalServerError, instanceLimitReachedErrorKey,
	)

	ErrBindingAlreadyExists = NewFailureResponse(
		errors.New(bindingExistsMsg), http.StatusConflict, bindingAlreadyExistsErrorKey,
	)

	ErrBindingDoesNotExist = NewFailureResponseBuilder(
		errors.New(bindingDoesntExistMsg), http.StatusGone, bindingMissingErrorKey,
	).WithEmptyResponse().Build()

	ErrBindingNotFound = NewFailureResponseBuilder(
		errors.New(bindingNotFoundMsg), http.StatusNotFound, bindingNotFoundErrorKey,
	).WithEmptyResponse().Build()

	ErrAsyncRequired = NewFailureResponseBuilder(
		errors.New(asyncRequiredMsg), http.StatusUnprocessableEntity, asyncRequiredKey,
	).WithErrorKey("AsyncRequired").Build()

	ErrPlanChangeNotSupported = NewFailureResponseBuilder(
		errors.New(planChangeUnsupportedMsg), http.StatusUnprocessableEntity, planChangeNotSupportedKey,
	).WithErrorKey("PlanChangeNotSupported").Build()

	ErrRawParamsInvalid = NewFailureResponse(
		errors.New(rawInvalidParamsMsg), http.StatusUnprocessableEntity, invalidRawParamsKey,
	)

	ErrAppGuidNotProvided = NewFailureResponse(
		errors.New(appGuidMissingMsg), http.StatusUnprocessableEntity, appGuidNotProvidedErrorKey,
	)

	ErrPlanQuotaExceeded    = errors.New(servicePlanQuotaExceededMsg)
	ErrServiceQuotaExceeded = errors.New(serviceQuotaExceededMsg)

	ErrConcurrentInstanceAccess = NewFailureResponseBuilder(
		errors.New(concurrentInstanceAccessMsg), http.StatusUnprocessableEntity, concurrentAccessKey,
	).WithErrorKey("ConcurrencyError")
)
