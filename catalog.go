package brokerapi

import (
	"encoding/json"
)

type Service struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Bindable        bool                    `json:"bindable"`
	Tags            []string                `json:"tags,omitempty"`
	PlanUpdatable   bool                    `json:"plan_updateable"`
	Plans           []ServicePlan           `json:"plans"`
	Requires        []RequiredPermission    `json:"requires,omitempty"`
	Metadata        *ServiceMetadata        `json:"metadata,omitempty"`
	DashboardClient *ServiceDashboardClient `json:"dashboard_client,omitempty"`
}

type ServiceDashboardClient struct {
	ID          string `json:"id"`
	Secret      string `json:"secret"`
	RedirectURI string `json:"redirect_uri"`
}

type ServicePlan struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Free        *bool                `json:"free,omitempty"`
	Bindable    *bool                `json:"bindable,omitempty"`
	Metadata    *ServicePlanMetadata `json:"metadata,omitempty"`
	Schemas     *ServiceSchemas      `json:"schemas,omitempty"`
}

type ServiceSchemas struct {
	Instance ServiceInstanceSchema `json:"service_instance,omitempty"`
	Binding  ServiceBindingSchema  `json:"service_binding,omitempty"`
}

type ServiceInstanceSchema struct {
	Create Schema `json:"create,omitempty"`
	Update Schema `json:"update,omitempty"`
}

type ServiceBindingSchema struct {
	Create Schema `json:"create,omitempty"`
}

type Schema struct {
	Schema interface{} `json:"parameters,omitempty"`
}

type ServicePlanMetadata struct {
	DisplayName        string            `json:"displayName,omitempty"`
	Bullets            []string          `json:"bullets,omitempty"`
	Costs              []ServicePlanCost `json:"costs,omitempty"`
	AdditionalMetadata map[string]interface{}
}

type ServicePlanCost struct {
	Amount map[string]float64 `json:"amount"`
	Unit   string             `json:"unit"`
}

type ServiceMetadata struct {
	DisplayName         string `json:"displayName,omitempty"`
	ImageUrl            string `json:"imageUrl,omitempty"`
	LongDescription     string `json:"longDescription,omitempty"`
	ProviderDisplayName string `json:"providerDisplayName,omitempty"`
	DocumentationUrl    string `json:"documentationUrl,omitempty"`
	SupportUrl          string `json:"supportUrl,omitempty"`
	Shareable           *bool  `json:"shareable,omitempty"`
	AdditionalMetadata  map[string]interface{}
}

func FreeValue(v bool) *bool {
	return &v
}

func BindableValue(v bool) *bool {
	return &v
}

type RequiredPermission string

const (
	PermissionRouteForwarding = RequiredPermission("route_forwarding")
	PermissionSyslogDrain     = RequiredPermission("syslog_drain")
	PermissionVolumeMount     = RequiredPermission("volume_mount")
)

func (spm ServicePlanMetadata) MarshalJSON() ([]byte, error) {
	type Alias ServicePlanMetadata
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&spm),
	}

	b, _ := json.Marshal(aux)

	m := spm.AdditionalMetadata

	json.Unmarshal(b, &m)

	delete(m, "AdditionalMetadata")

	return json.Marshal(m)
}

func (spm *ServicePlanMetadata) UnmarshalJSON(data []byte) error {
	type Alias ServicePlanMetadata
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(spm),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	additionalMetadata := map[string]interface{}{}
	if err := json.Unmarshal(data, &additionalMetadata); err != nil {
		return err
	}

	delete(additionalMetadata, "displayName")
	delete(additionalMetadata, "bullets")
	delete(additionalMetadata, "costs")

	spm.AdditionalMetadata = additionalMetadata

	return nil
}

func (sm ServiceMetadata) MarshalJSON() ([]byte, error) {
	type Alias ServiceMetadata
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&sm),
	}

	b, _ := json.Marshal(aux)

	m := sm.AdditionalMetadata

	json.Unmarshal(b, &m)

	delete(m, "AdditionalMetadata")

	return json.Marshal(m)
}

func (sm *ServiceMetadata) UnmarshalJSON(data []byte) error {
	type Alias ServiceMetadata
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sm),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	additionalMetadata := map[string]interface{}{}
	if err := json.Unmarshal(data, &additionalMetadata); err != nil {
		return err
	}

	delete(additionalMetadata, "displayName")
	delete(additionalMetadata, "imageUrl")
	delete(additionalMetadata, "longDescription")
	delete(additionalMetadata, "providerDisplayName")
	delete(additionalMetadata, "documentationUrl")
	delete(additionalMetadata, "supportUrl")
	delete(additionalMetadata, "shareable")

	sm.AdditionalMetadata = additionalMetadata

	return nil
}
