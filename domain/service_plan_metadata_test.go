package domain_test

import (
	"encoding/json"
	"reflect"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
)

var _ = Describe("ServicePlanMetadata", func() {
	Describe("JSON encoding", func() {
		It("uses the correct keys", func() {
			metadata := domain.ServicePlanMetadata{
				Bullets:     []string{"test"},
				DisplayName: "Some display name",
			}
			jsonString := `{"bullets":["test"],"displayName":"Some display name"}`

			Expect(json.Marshal(metadata)).To(MatchJSON(jsonString))
		})

		It("encodes the AdditionalMetadata fields in the metadata fields", func() {
			metadata := domain.ServicePlanMetadata{
				Bullets:     []string{"hello", "its me"},
				DisplayName: "name",
				AdditionalMetadata: map[string]interface{}{
					"foo": "bar",
					"baz": 1,
				},
			}
			jsonString := `{
					"bullets":["hello", "its me"],
					"displayName":"name",
					"foo": "bar",
					"baz": 1
				}`

			Expect(json.Marshal(metadata)).To(MatchJSON(jsonString))

			By("not mutating the AdditionalMetadata during custom JSON marshalling")
			Expect(len(metadata.AdditionalMetadata)).To(Equal(2))
		})

		It("it can marshal same structure in parallel requests", func() {
			metadata := domain.ServicePlanMetadata{
				Bullets:     []string{"hello", "its me"},
				DisplayName: "name",
				AdditionalMetadata: map[string]interface{}{
					"foo": "bar",
					"baz": 1,
				},
			}
			jsonString := `{
					"bullets":["hello", "its me"],
					"displayName":"name",
					"foo": "bar",
					"baz": 1
				}`

			var wg sync.WaitGroup
			wg.Add(2)

			for i := 0; i < 2; i++ {
				go func() {
					defer wg.Done()
					defer GinkgoRecover()

					Expect(json.Marshal(metadata)).To(MatchJSON(jsonString))
				}()
			}
			wg.Wait()
		})

		It("returns an error when additional metadata is not marshallable", func() {
			metadata := domain.ServicePlanMetadata{
				Bullets:     []string{"hello", "its me"},
				DisplayName: "name",
				AdditionalMetadata: map[string]interface{}{
					"foo": make(chan int, 0),
				},
			}
			_, err := json.Marshal(metadata)
			Expect(err).To(MatchError(ContainSubstring("unmarshallable content in AdditionalMetadata")))
		})
	})

	Describe("JSON decoding", func() {
		It("sets the AdditionalMetadata from unrecognized fields", func() {
			metadata := domain.ServicePlanMetadata{}
			jsonString := `{"foo":["test"],"bar":"Some display name"}`

			err := json.Unmarshal([]byte(jsonString), &metadata)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata.AdditionalMetadata["foo"]).To(Equal([]interface{}{"test"}))
			Expect(metadata.AdditionalMetadata["bar"]).To(Equal("Some display name"))
		})

		It("does not include convention fields into additional metadata", func() {
			metadata := domain.ServicePlanMetadata{}
			jsonString := `{"bullets":["test"],"displayName":"Some display name", "costs": [{"amount": {"usd": 649.0},"unit": "MONTHLY"}]}`

			err := json.Unmarshal([]byte(jsonString), &metadata)
			Expect(err).NotTo(HaveOccurred())
			Expect(metadata.AdditionalMetadata).To(BeNil())
		})
	})

	Describe("GetJsonNames", func() {
		It("Reflects JSON names from struct", func() {
			type Example1 struct {
				Foo int    `json:"foo"`
				Bar string `yaml:"hello" json:"bar,omitempty"`
				Qux float64
			}

			s := Example1{}
			Expect(brokerapi.GetJsonNames(reflect.ValueOf(&s).Elem())).To(
				ConsistOf([]string{"foo", "bar", "Qux"}))
		})
	})
})
