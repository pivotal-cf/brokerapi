package brokerapi_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/brokerapi/matchers"

	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Catalog", func() {
	Describe("Service", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				service := brokerapi.Service{
					ID:          "ID-1",
					Name:        "Cassandra",
					Description: "A Cassandra Plan",
					Bindable:    true,
					Plans:       []brokerapi.ServicePlan{},
					Metadata:    brokerapi.ServiceMetadata{},
					Tags:        []string{},
				}
				json := `{"id":"ID-1","name":"Cassandra","description":"A Cassandra Plan","bindable":true,"plans":[],"metadata":{"displayName":"","longDescription":"","documentationUrl":"","supportUrl":"","listing":{"blurb":"","imageUrl":""},"provider":{"name":""}},"tags":[]}`

				Expect(service).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServicePlan", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				plan := brokerapi.ServicePlan{
					ID:          "ID-1",
					Name:        "Cassandra",
					Description: "A Cassandra Plan",
					Metadata: brokerapi.ServicePlanMetadata{
						Bullets: []string{},
					},
				}
				json := `{"id":"ID-1","name":"Cassandra","description":"A Cassandra Plan","metadata":{"bullets":[],"displayName":""}}`

				Expect(plan).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServicePlanMetadata", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				metadata := brokerapi.ServicePlanMetadata{
					Bullets:     []string{},
					DisplayName: "Some display name",
				}
				json := `{"bullets":[],"displayName":"Some display name"}`

				Expect(metadata).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServiceMetadata", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				metadata := brokerapi.ServiceMetadata{
					DisplayName:      "Cassandra",
					LongDescription:  "A long description of Cassandra",
					DocumentationUrl: "",
					SupportUrl:       "",
					Listing:          brokerapi.ServiceMetadataListing{},
					Provider:         brokerapi.ServiceMetadataProvider{},
				}
				json := `{"displayName":"Cassandra","longDescription":"A long description of Cassandra","documentationUrl":"","supportUrl":"","listing":{"blurb":"","imageUrl":""},"provider":{"name":""}}`

				Expect(metadata).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServiceMetadataListing", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				listing := brokerapi.ServiceMetadataListing{
					Blurb:    "Blurb",
					ImageUrl: "foo",
				}
				json := `{"blurb":"Blurb","imageUrl":"foo"}`

				Expect(listing).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServiceMetadataProvider", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				provider := brokerapi.ServiceMetadataProvider{
					Name: "Pivotal",
				}
				json := `{"name":"Pivotal"}`

				Expect(provider).To(MarshalToJSON(json))
			})
		})
	})
})
