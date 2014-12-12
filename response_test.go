package brokerapi_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/brokerapi/matchers"

	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Catalog Response", func() {
	Describe("JSON encoding", func() {
		It("has a list of services", func() {
			catalogResponse := brokerapi.CatalogResponse{
				Services: []brokerapi.Service{},
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
				provisioningResponse := brokerapi.ProvisioningResponse{}
				json := `{}`

				Expect(provisioningResponse).To(MarshalToJSON(json))
			})
		})

		Context("when the dashboard URL is present", func() {
			It("returns it in the JSON", func() {
				provisioningResponse := brokerapi.ProvisioningResponse{
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
			bindingResponse := brokerapi.BindingResponse{}
			json := `{"credentials":null}`

			Expect(bindingResponse).To(MarshalToJSON(json))
		})
	})
})

var _ = Describe("Error Response", func() {
	Describe("JSON encoding", func() {
		It("has a description field", func() {
			errorResponse := brokerapi.ErrorResponse{
				Description: "a bad thing happened",
			}
			json := `{"description":"a bad thing happened"}`

			Expect(errorResponse).To(MarshalToJSON(json))
		})
	})
})
