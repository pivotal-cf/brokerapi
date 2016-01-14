package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/matchers"
)

func MarshalToJSON(expectedJSON string) gomega.OmegaMatcher {
	return &marshalToJSONMatcher{
		expectedJSON: expectedJSON,
		jsonMatcher:  matchers.MatchJSONMatcher{expectedJSON},
	}
}

type marshalToJSONMatcher struct {
	expectedJSON string
	jsonMatcher  matchers.MatchJSONMatcher
}

func (m *marshalToJSONMatcher) Match(actual interface{}) (success bool, err error) {
	bytes, err := json.Marshal(actual)

	if err != nil {
		return false, err
	}

	return m.jsonMatcher.Match(bytes)
}

func (m *marshalToJSONMatcher) FailureMessage(actual interface{}) (message string) {
	bytes, _ := json.Marshal(actual)

	return fmt.Sprintf("expected\n %s\n to equal\n %s", string(bytes), m.expectedJSON)
}

func (m *marshalToJSONMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	bytes, _ := json.Marshal(actual)
	return fmt.Sprintf("expected\n %s\n to not equal\n %s", string(bytes), m.expectedJSON)
}
