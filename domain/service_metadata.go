package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type ServiceMetadata struct {
	DisplayName         string `json:"displayName,omitempty"`
	ImageUrl            string `json:"imageUrl,omitempty"`
	LongDescription     string `json:"longDescription,omitempty"`
	ProviderDisplayName string `json:"providerDisplayName,omitempty"`
	DocumentationUrl    string `json:"documentationUrl,omitempty"`
	SupportUrl          string `json:"supportUrl,omitempty"`
	Shareable           *bool  `json:"shareable,omitempty"`
	AdditionalMetadata  map[string]any
}

func (sm ServiceMetadata) MarshalJSON() ([]byte, error) {
	type Alias ServiceMetadata

	b, err := json.Marshal(Alias(sm))
	if err != nil {
		return nil, fmt.Errorf("unmarshallable content in AdditionalMetadata: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	delete(m, additionalMetadataName)

	for k, v := range sm.AdditionalMetadata {
		m[k] = v
	}
	return json.Marshal(m)
}

func (sm *ServiceMetadata) UnmarshalJSON(data []byte) error {
	type Alias ServiceMetadata

	if err := json.Unmarshal(data, (*Alias)(sm)); err != nil {
		return err
	}

	additionalMetadata := map[string]any{}
	if err := json.Unmarshal(data, &additionalMetadata); err != nil {
		return err
	}

	for _, jsonName := range GetJsonNames(reflect.ValueOf(sm).Elem()) {
		if jsonName == additionalMetadataName {
			continue
		}
		delete(additionalMetadata, jsonName)
	}

	if len(additionalMetadata) > 0 {
		sm.AdditionalMetadata = additionalMetadata
	}
	return nil
}
