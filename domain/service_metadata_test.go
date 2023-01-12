package domain_test

import (
	"encoding/json"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v9/domain"
)

var _ = Describe("ServiceMetadata", func() {
	Describe("ServiceMetadata", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				shareable := true
				metadata := domain.ServiceMetadata{
					DisplayName:         "Cassandra",
					LongDescription:     "A long description of Cassandra",
					DocumentationUrl:    "doc",
					SupportUrl:          "support",
					ImageUrl:            "image",
					ProviderDisplayName: "display",
					Shareable:           &shareable,
				}
				jsonString := `{
					"displayName":"Cassandra",
					"longDescription":"A long description of Cassandra",
					"documentationUrl":"doc",
					"supportUrl":"support",
					"imageUrl":"image",
					"providerDisplayName":"display",
					"shareable":true
				}`

				Expect(json.Marshal(metadata)).To(MatchJSON(jsonString))
			})

			It("encodes the AdditionalMetadata fields in the metadata fields", func() {
				metadata := domain.ServiceMetadata{
					DisplayName: "name",
					AdditionalMetadata: map[string]interface{}{
						"foo": "bar",
						"baz": 1,
					},
				}
				jsonString := `{
					"displayName":"name",
					"foo": "bar",
					"baz": 1
				}`

				Expect(json.Marshal(metadata)).To(MatchJSON(jsonString))

				By("not mutating the AdditionalMetadata during custom JSON marshalling")
				Expect(len(metadata.AdditionalMetadata)).To(Equal(2))
			})

			It("it can marshal same structure in parallel requests", func() {
				metadata := domain.ServiceMetadata{
					DisplayName: "name",
					AdditionalMetadata: map[string]interface{}{
						"foo": "bar",
						"baz": 1,
					},
				}
				jsonString := `{
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
				metadata := domain.ServiceMetadata{
					DisplayName: "name",
					AdditionalMetadata: map[string]interface{}{
						"foo": make(chan int),
					},
				}
				_, err := json.Marshal(metadata)
				Expect(err).To(MatchError(ContainSubstring("unmarshallable content in AdditionalMetadata")))
			})
		})

		Describe("JSON decoding", func() {
			It("sets the AdditionalMetadata from unrecognized fields", func() {
				metadata := domain.ServiceMetadata{}
				jsonString := `{"foo":["test"],"bar":"Some display name"}`

				err := json.Unmarshal([]byte(jsonString), &metadata)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata.AdditionalMetadata["foo"]).To(Equal([]interface{}{"test"}))
				Expect(metadata.AdditionalMetadata["bar"]).To(Equal("Some display name"))
			})

			It("does not include convention fields into additional metadata", func() {
				metadata := domain.ServiceMetadata{}
				jsonString := `{
					"displayName":"Cassandra",
					"longDescription":"A long description of Cassandra",
					"documentationUrl":"doc",
					"supportUrl":"support",
					"imageUrl":"image",
					"providerDisplayName":"display",
					"shareable":true
				}`
				err := json.Unmarshal([]byte(jsonString), &metadata)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata.AdditionalMetadata).To(BeNil())
			})
		})
	})
})
