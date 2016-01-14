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
					ID:            "ID-1",
					Name:          "Cassandra",
					Description:   "A Cassandra Plan",
					Bindable:      true,
					Plans:         []brokerapi.ServicePlan{},
					Metadata:      &brokerapi.ServiceMetadata{},
					Tags:          []string{"test"},
					PlanUpdatable: true,
					DashboardClient: &brokerapi.ServiceDashboardClient{
						ID:          "Dashboard ID",
						Secret:      "dashboardsecret",
						RedirectURI: "the.dashboa.rd",
					},
				}
				json := `{
					"id":"ID-1",
				  	"name":"Cassandra",
					"description":"A Cassandra Plan",
					"bindable":true,
					"plan_updateable":true,
					"tags":["test"],
					"plans":[],
					"dashboard_client":{
						"id":"Dashboard ID",
						"secret":"dashboardsecret",
						"redirect_uri":"the.dashboa.rd"
					},
					"metadata":{

					}
				}`
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
					Free:        brokerapi.FreeValue(true),
					Metadata: &brokerapi.ServicePlanMetadata{
						Bullets:     []string{"hello", "its me"},
						DisplayName: "name",
					},
				}
				json := `{
					"id":"ID-1",
					"name":"Cassandra",
					"description":"A Cassandra Plan",
					"free": true,
					"metadata":{
						"bullets":["hello", "its me"],
						"displayName":"name"
					}
				}`

				Expect(plan).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServicePlanMetadata", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				metadata := brokerapi.ServicePlanMetadata{
					Bullets:     []string{"test"},
					DisplayName: "Some display name",
				}
				json := `{"bullets":["test"],"displayName":"Some display name"}`

				Expect(metadata).To(MarshalToJSON(json))
			})
		})
	})

	Describe("ServiceMetadata", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				metadata := brokerapi.ServiceMetadata{
					DisplayName:         "Cassandra",
					LongDescription:     "A long description of Cassandra",
					DocumentationUrl:    "doc",
					SupportUrl:          "support",
					ImageUrl:            "image",
					ProviderDisplayName: "display",
				}
				json := `{
					"displayName":"Cassandra",
					"longDescription":"A long description of Cassandra",
					"documentationUrl":"doc",
					"supportUrl":"support",
					"imageUrl":"image",
					"providerDisplayName":"display"
				}`

				Expect(metadata).To(MarshalToJSON(json))
			})
		})
	})
})
