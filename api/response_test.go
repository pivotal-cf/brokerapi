package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/go-service-broker/api/matchers"

	"github.com/pivotal-cf/go-service-broker/api"
)

var _ = Describe("Catalog Response", func() {
	Describe("JSON encoding", func() {
		It("has a list of services", func() {
			catalogResponse := api.CatalogResponse{
				Services: []api.Service{},
			}
			json := `{"services":[]}`

			Expect(catalogResponse).To(MarshalToJSON(json))
		})
	})
})

var _ = Describe("Provisioning Response", func() {
	Describe("JSON encoding", func() {
		Context("when the dashboard URL is not present", func() {
			It("does not return it in the JSON", func() {
				provisioningResponse := api.ProvisioningResponse{}
				json := `{}`

				Expect(provisioningResponse).To(MarshalToJSON(json))
			})
		})

		Context("when the dashboard URL is present", func() {
			It("returns it in the JSON", func() {
				provisioningResponse := api.ProvisioningResponse{
					DashboardURL: "http://example.com/broker",
				}
				json := `{"dashboard_url":"http://example.com/broker"}`

				Expect(provisioningResponse).To(MarshalToJSON(json))
			})
		})
	})
})

var _ = Describe("Binding Response", func() {
	Describe("JSON encoding", func() {
		It("has a credentials object", func() {
			bindingResponse := api.BindingResponse{}
			json := `{"credentials":null}`

			Expect(bindingResponse).To(MarshalToJSON(json))
		})
	})
})

var _ = Describe("Error Response", func() {
	Describe("JSON encoding", func() {
		It("has a description field", func() {
			errorResponse := api.ErrorResponse{
				Description: "a bad thing happened",
			}
			json := `{"description":"a bad thing happened"}`

			Expect(errorResponse).To(MarshalToJSON(json))
		})
	})
})
