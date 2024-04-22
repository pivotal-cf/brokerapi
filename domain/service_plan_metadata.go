package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type ServicePlanMetadata struct {
	DisplayName        string            `json:"displayName,omitempty"`
	Bullets            []string          `json:"bullets,omitempty"`
	Costs              []ServicePlanCost `json:"costs,omitempty"`
	AdditionalMetadata map[string]any
}

type ServicePlanCost struct {
	Amount map[string]float64 `json:"amount"`
	Unit   string             `json:"unit"`
}

func (spm *ServicePlanMetadata) UnmarshalJSON(data []byte) error {
	type Alias ServicePlanMetadata

	if err := json.Unmarshal(data, (*Alias)(spm)); err != nil {
		return err
	}

	additionalMetadata := map[string]any{}
	if err := json.Unmarshal(data, &additionalMetadata); err != nil {
		return err
	}

	s := reflect.ValueOf(spm).Elem()
	for _, jsonName := range GetJsonNames(s) {
		if jsonName == additionalMetadataName {
			continue
		}
		delete(additionalMetadata, jsonName)
	}

	if len(additionalMetadata) > 0 {
		spm.AdditionalMetadata = additionalMetadata
	}
	return nil
}

func (spm ServicePlanMetadata) MarshalJSON() ([]byte, error) {
	type Alias ServicePlanMetadata

	b, err := json.Marshal(Alias(spm))
	if err != nil {
		return nil, fmt.Errorf("unmarshallable content in AdditionalMetadata: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	delete(m, additionalMetadataName)

	for k, v := range spm.AdditionalMetadata {
		m[k] = v
	}

	return json.Marshal(m)
}

func GetJsonNames(s reflect.Value) (res []string) {
	valType := s.Type()
	for i := 0; i < s.NumField(); i++ {
		field := valType.Field(i)
		tag := field.Tag
		jsonVal := tag.Get("json")
		if jsonVal != "" {
			components := strings.Split(jsonVal, ",")
			jsonName := components[0]
			res = append(res, jsonName)
		} else {
			res = append(res, field.Name)
		}
	}
	return res
}
