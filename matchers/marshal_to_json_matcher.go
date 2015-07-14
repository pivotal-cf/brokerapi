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

func (m *marshalToJSONMatcher) Match(actual interface{}) (success bool, err error) {
	bytes, err := json.Marshal(actual)
	if err != nil {
		return false, err
	}

	actualJSON := string(bytes)
	if actualJSON == m.expectedJSON {
		return true, nil
	}

	return false, nil
}

func (m *marshalToJSONMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected %s to equal %s", actual, m.expectedJSON)
}

func (m *marshalToJSONMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected %s not to equal %s", actual, m.expectedJSON)
}
