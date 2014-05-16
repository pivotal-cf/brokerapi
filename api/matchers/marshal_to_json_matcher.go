package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/gomega"
)

func MarshalToJSON(expectedJSON string) gomega.OmegaMatcher {
	return &marshalToJSONMatcher{
		expectedJSON: expectedJSON,
	}
}

type marshalToJSONMatcher struct {
	expectedJSON string
}

func (m *marshalToJSONMatcher) Match(actual interface{}) (success bool, message string, err error) {
	bytes, err := json.Marshal(actual)
	if err != nil {
		return false, "Could not marshal the object to JSON", err
	}

	actualJSON := string(bytes)
	if actualJSON == m.expectedJSON {
		return true, fmt.Sprintf("expected %s not to equal %s", actualJSON, m.expectedJSON), nil
	}

	return false, fmt.Sprintf("expected %s to equal %s", actualJSON, m.expectedJSON), nil
}
