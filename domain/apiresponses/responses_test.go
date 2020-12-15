package apiresponses_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
)

var _ = Describe("Catalog Response", func() {
	Describe("JSON encoding", func() {
		It("has a list of services", func() {
			catalogResponse := apiresponses.CatalogResponse{
				Services: []domain.Service{},
			}
			jsonString := `{"services":[]}`

			Expect(json.Marshal(catalogResponse)).To(MatchJSON(jsonString))
		})
	})
})

var _ = Describe("Provisioning Response", func() {
	Describe("JSON encoding", func() {
		Context("when the dashboard URL is not present", func() {
			It("does not return it in the JSON", func() {
				provisioningResponse := apiresponses.ProvisioningResponse{}
				jsonString := `{}`

				Expect(json.Marshal(provisioningResponse)).To(MatchJSON(jsonString))
			})
		})

		Context("when the dashboard URL is present", func() {
			It("returns it in the JSON", func() {
				provisioningResponse := apiresponses.ProvisioningResponse{
					DashboardURL: "http://example.com/broker",
				}
				jsonString := `{"dashboard_url":"http://example.com/broker"}`

				Expect(json.Marshal(provisioningResponse)).To(MatchJSON(jsonString))
			})
		})

		Context("when the metadata is present", func() {
			It("returns it in the JSON", func() {
				provisioningResponse := apiresponses.ProvisioningResponse{
					Metadata: domain.InstanceMetadata{
						Labels:     map[string]string{"key1": "value1"},
						Attributes: map[string]string{"key1": "value1"},
					},
				}
				jsonString := `{"metadata":{"labels":{"key1":"value1"}, "attributes":{"key1":"value1"}}}`

				Expect(json.Marshal(provisioningResponse)).To(MatchJSON(jsonString))
			})
		})
	})
})

var _ = Describe("Update Response", func() {
	Describe("JSON encoding", func() {
		Context("when the dashboard URL is not present", func() {
			It("does not return it in the JSON", func() {
				updateResponse := apiresponses.UpdateResponse{}
				jsonString := `{}`

				Expect(json.Marshal(updateResponse)).To(MatchJSON(jsonString))
			})
		})

		Context("when the dashboard URL is present", func() {
			It("returns it in the JSON", func() {
				updateResponse := apiresponses.UpdateResponse{
					DashboardURL: "http://example.com/broker_updated",
				}
				jsonString := `{"dashboard_url":"http://example.com/broker_updated"}`

				Expect(json.Marshal(updateResponse)).To(MatchJSON(jsonString))
			})
		})
	})
})

var _ = Describe("Error Response", func() {
	Describe("JSON encoding", func() {
		It("has a description field", func() {
			errorResponse := apiresponses.ErrorResponse{
				Description: "a bad thing happened",
			}
			jsonString := `{"description":"a bad thing happened"}`

			Expect(json.Marshal(errorResponse)).To(MatchJSON(jsonString))
		})
	})
})
