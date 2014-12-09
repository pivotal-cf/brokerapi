package api_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/drewolson/testflight"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/go-service-broker/api"
)

var _ = Describe("Service Broker API", func() {
	var fakeServiceBroker *api.FakeServiceBroker
	var brokerAPI http.Handler
	var brokerLogger *lagertest.TestLogger

	makeInstanceProvisioningRequest := func(instanceID string, params map[string]string) *testflight.Response {
		response := &testflight.Response{}
		testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
			path := "/v2/service_instances/" + instanceID

			request, _ := http.NewRequest("PUT", path, strings.NewReader(`{
          "planID":           "`+params["planID"]+`",
          "organizationGUID": "`+params["organizationGUID"]+`",
          "spaceGUID":        "`+params["spaceGUID"]+`"
      }`))
			request.Header.Add("Content-Type", "application/json")
			request.SetBasicAuth("username", "password")

			response = r.Do(request)
		})
		return response
	}

	lastLogLine := func() lager.LogFormat {
		return brokerLogger.Logs()[0]
	}

	BeforeEach(func() {
		os.Setenv("BROKER_USER", "username")
		os.Setenv("BROKER_PASSWORD", "password")

		fakeServiceBroker = &api.FakeServiceBroker{
			InstanceLimit: 3,
		}
		brokerLogger = lagertest.NewTestLogger("broker-api")
		brokerAPI = api.New(fakeServiceBroker, nullLogger(), brokerLogger)
	})

	Describe("authentication", func() {
		makeRequestWithoutAuth := func() *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				request, _ := http.NewRequest("GET", "/v2/catalog", nil)
				response = r.Do(request)
			})
			return response
		}

		makeRequestWithAuth := func(username string, password string) *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				request, _ := http.NewRequest("GET", "/v2/catalog", nil)
				request.SetBasicAuth(username, password)

				response = r.Do(request)
			})
			return response
		}

		makeRequestWithUnrecognizedAuth := func() *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				request, _ := http.NewRequest("GET", "/v2/catalog", nil)
				// dXNlcm5hbWU6cGFzc3dvcmQ= is base64 encoding of 'username:password',
				// ie, a correctly encoded basic authorization header
				request.Header["Authorization"] = []string{"NOTBASIC dXNlcm5hbWU6cGFzc3dvcmQ="}

				response = r.Do(request)
			})
			return response
		}

		It("returns 401 when the authorization header has an incorrect password", func() {
			response := makeRequestWithAuth("username", "fake_password")
			Expect(response.StatusCode).To(Equal(401))
		})

		It("returns 401 when the authorization header has an incorrect username", func() {
			response := makeRequestWithAuth("fake_username", "password")
			Expect(response.StatusCode).To(Equal(401))
		})

		It("returns 401 when there is no authorization header", func() {
			response := makeRequestWithoutAuth()
			Expect(response.StatusCode).To(Equal(401))
		})

		It("returns 401 when there is a unrecognized authorization header", func() {
			response := makeRequestWithUnrecognizedAuth()
			Expect(response.StatusCode).To(Equal(401))
		})
	})

	Describe("catalog endpoint", func() {
		makeCatalogRequest := func() *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				request, _ := http.NewRequest("GET", "/v2/catalog", nil)
				request.SetBasicAuth("username", "password")

				response = r.Do(request)
			})
			return response
		}

		It("returns a 200", func() {
			response := makeCatalogRequest()
			Expect(response.StatusCode).To(Equal(200))
		})

		It("returns valid catalog json", func() {
			response := makeCatalogRequest()
			Expect(response.Body).To(MatchJSON(fixture("catalog.json")))
		})
	})

	Describe("instance lifecycle endpoint", func() {
		makeInstanceDeprovisioningRequest := func(instanceID string) *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				path := "/v2/service_instances/" + instanceID
				request, _ := http.NewRequest("DELETE", path, strings.NewReader(""))
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")

				response = r.Do(request)

			})
			return response
		}

		Describe("provisioning", func() {
			var instanceID string
			var params map[string]string

			BeforeEach(func() {
				instanceID = uniqueInstanceID()
				params = map[string]string{
					"planID":           "plan-id",
					"organizationGUID": "organization-guid",
					"spaceGUID":        "space-guid",
				}
			})

			It("calls Provision on the service broker with all params", func() {
				makeInstanceProvisioningRequest(instanceID, params)
				Expect(fakeServiceBroker.Params).To(Equal(params))
			})

			It("calls Provision on the service broker with the instance id", func() {
				makeInstanceProvisioningRequest(instanceID, params)
				Expect(fakeServiceBroker.ProvisionedInstanceIDs).To(ContainElement(instanceID))
			})

			Context("when the instance does not exist", func() {
				It("returns a 201", func() {
					response := makeInstanceProvisioningRequest(instanceID, params)
					Expect(response.StatusCode).To(Equal(201))
				})

				It("returns json with a dashboard_url field", func() {
					response := makeInstanceProvisioningRequest(instanceID, params)
					Expect(response.Body).To(MatchJSON(fixture("provisioning.json")))
				})

				Context("when the instance limit has been reached", func() {
					BeforeEach(func() {
						for i := 0; i < fakeServiceBroker.InstanceLimit; i++ {
							makeInstanceProvisioningRequest(uniqueInstanceID(), params)
						}
					})

					It("returns a 500", func() {
						response := makeInstanceProvisioningRequest(instanceID, params)
						Expect(response.StatusCode).To(Equal(500))
					})

					It("returns json with a description field and a useful error message", func() {
						response := makeInstanceProvisioningRequest(instanceID, params)
						Expect(response.Body).To(MatchJSON(fixture("instance_limit_error.json")))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, params)

						Expect(lastLogLine().Message).To(ContainSubstring("provision.instance-limit-reached"))
						Expect(lastLogLine().Data["error"]).To(ContainSubstring("instance limit for this service has been reached"))
					})
				})

				Context("when an unexpected error occurs", func() {
					BeforeEach(func() {
						fakeServiceBroker.ProvisionError = errors.New("broker failed")
					})

					It("returns a 500", func() {
						response := makeInstanceProvisioningRequest(instanceID, params)
						Expect(response.StatusCode).To(Equal(500))
					})

					It("returns json with a description field and a useful error message", func() {
						response := makeInstanceProvisioningRequest(instanceID, params)
						Expect(response.Body).To(MatchJSON(fixture("unexpected_error.json")))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, params)
						Expect(lastLogLine().Message).To(ContainSubstring("provision.unknown-error"))
						Expect(lastLogLine().Data["error"]).To(ContainSubstring("broker failed"))
					})
				})

			})

			Context("when the instance already exists", func() {
				BeforeEach(func() {
					makeInstanceProvisioningRequest(instanceID, params)
				})

				It("returns a 409", func() {
					response := makeInstanceProvisioningRequest(instanceID, params)
					Expect(response.StatusCode).To(Equal(409))
				})

				It("returns an empty JSON object", func() {
					response := makeInstanceProvisioningRequest(instanceID, params)
					Expect(response.Body).To(Equal(`{}`))
				})

				It("logs an appropriate error", func() {
					makeInstanceProvisioningRequest(instanceID, params)
					Expect(lastLogLine().Message).To(ContainSubstring("provision.instance-already-exists"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("instance already exists"))
				})
			})
		})

		Describe("deprovisioning", func() {
			It("calls Deprovision on the service broker with the instance id", func() {
				instanceID := uniqueInstanceID()
				makeInstanceDeprovisioningRequest(instanceID)
				Expect(fakeServiceBroker.DeprovisionedInstanceIDs).To(ContainElement(instanceID))
			})

			Context("when the instance exists", func() {
				var instanceID string
				var params map[string]string

				BeforeEach(func() {
					instanceID = uniqueInstanceID()
					params = map[string]string{
						"planID":           "plan-id",
						"organizationGUID": "organization-guid",
						"spaceGUID":        "space-guid",
					}
					makeInstanceProvisioningRequest(instanceID, params)
				})

				It("returns a 200", func() {
					response := makeInstanceDeprovisioningRequest(instanceID)
					Expect(response.StatusCode).To(Equal(200))
				})

				It("returns an empty JSON object", func() {
					response := makeInstanceDeprovisioningRequest(instanceID)
					Expect(response.Body).To(MatchJSON(`{}`))
				})
			})

			Context("when the instance does not exist", func() {
				var instanceID string

				It("returns a 410", func() {
					response := makeInstanceDeprovisioningRequest(uniqueInstanceID())
					Expect(response.StatusCode).To(Equal(410))
				})

				It("returns an empty JSON object", func() {
					response := makeInstanceDeprovisioningRequest(uniqueInstanceID())
					Expect(response.Body).To(MatchJSON(`{}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeInstanceDeprovisioningRequest(instanceID)
					Expect(lastLogLine().Message).To(ContainSubstring("deprovision.instance-missing"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("instance does not exist"))
				})
			})
		})
	})

	Describe("binding lifecycle endpoint", func() {
		makeBindingRequest := func(instanceID string, bindingID string) *testflight.Response {
			response := &testflight.Response{}
			testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
				path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s",
					instanceID, bindingID)
				request, _ := http.NewRequest("PUT", path, strings.NewReader(""))
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")

				response = r.Do(request)
			})
			return response
		}

		Describe("binding", func() {
			Context("when the associated instance exists", func() {
				It("calls Bind on the service broker with the instance and binding ids", func() {
					instanceID := uniqueInstanceID()
					bindingID := uniqueBindingID()
					makeBindingRequest(instanceID, bindingID)
					Expect(fakeServiceBroker.BoundInstanceIDs).To(ContainElement(instanceID))
					Expect(fakeServiceBroker.BoundBindingIDs).To(ContainElement(bindingID))
				})

				It("returns the credentials returned by Bind", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.Body).To(MatchJSON(fixture("binding.json")))
				})

				It("returns a 201", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.StatusCode).To(Equal(201))
				})
			})

			Context("when the associated instance does not exist", func() {
				var instanceID string

				BeforeEach(func() {
					fakeServiceBroker.BindError = api.ErrInstanceDoesNotExist
				})

				It("returns a 404", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.StatusCode).To(Equal(404))
				})

				It("returns an error JSON object", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.Body).To(MatchJSON(`{"description":"instance does not exist"}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeBindingRequest(instanceID, uniqueBindingID())
					Expect(lastLogLine().Message).To(ContainSubstring("bind.instance-missing"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("instance does not exist"))
				})
			})

			Context("when the requested binding already exists", func() {
				var instanceID string

				BeforeEach(func() {
					fakeServiceBroker.BindError = api.ErrBindingAlreadyExists
				})

				It("returns a 409", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.StatusCode).To(Equal(409))
				})

				It("returns an error JSON object", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.Body).To(MatchJSON(`{"description":"binding already exists"}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeBindingRequest(instanceID, uniqueBindingID())
					makeBindingRequest(instanceID, uniqueBindingID())

					Expect(lastLogLine().Message).To(ContainSubstring("bind.binding-already-exists"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("binding already exists"))
				})
			})

			Context("when the binding returns an error", func() {
				BeforeEach(func() {
					fakeServiceBroker.BindError = errors.New("random error")
				})

				It("returns a generic 500 error response", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.StatusCode).To(Equal(500))
					Expect(response.Body).To(MatchJSON(`{"description":"random error"}`))
				})

				It("logs a detailed error message", func() {
					makeBindingRequest(uniqueInstanceID(), uniqueBindingID())

					Expect(lastLogLine().Message).To(ContainSubstring("bind.unknown-error"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("random error"))
				})
			})
		})

		Describe("unbinding", func() {
			makeUnbindingRequest := func(instanceID string, bindingID string) *testflight.Response {
				response := &testflight.Response{}
				testflight.WithServer(brokerAPI, func(r *testflight.Requester) {
					path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s",
						instanceID, bindingID)
					request, _ := http.NewRequest("DELETE", path, strings.NewReader(""))
					request.Header.Add("Content-Type", "application/json")
					request.SetBasicAuth("username", "password")

					response = r.Do(request)
				})
				return response
			}

			Context("when the associated instance exists", func() {
				var instanceID string
				var params map[string]string

				BeforeEach(func() {
					instanceID = uniqueInstanceID()
					params = map[string]string{
						"planID":           "plan-id",
						"organizationGUID": "organization-guid",
						"spaceGUID":        "space-guid",
					}
					makeInstanceProvisioningRequest(instanceID, params)
				})

				Context("and the binding exists", func() {
					var bindingID string

					BeforeEach(func() {
						bindingID = uniqueBindingID()
						makeBindingRequest(instanceID, bindingID)
					})

					It("returns a 200", func() {
						response := makeUnbindingRequest(instanceID, bindingID)
						Expect(response.StatusCode).To(Equal(200))
					})

					It("returns an empty JSON object", func() {
						response := makeUnbindingRequest(instanceID, bindingID)
						Expect(response.Body).To(Equal(`{}`))
					})
				})

				Context("but the binding does not exist", func() {
					It("returns a 410", func() {
						response := makeUnbindingRequest(instanceID, "does-not-exist")
						Expect(response.StatusCode).To(Equal(410))
					})

					It("logs an appropriate error message", func() {
						makeUnbindingRequest(instanceID, "does-not-exist")

						Expect(lastLogLine().Message).To(ContainSubstring("bind.binding-missing"))
						Expect(lastLogLine().Data["error"]).To(ContainSubstring("binding does not exist"))
					})
				})
			})

			Context("when the associated instance does not exist", func() {
				var instanceID string

				It("returns a 404", func() {
					response := makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.StatusCode).To(Equal(404))
				})

				It("returns an empty JSON object", func() {
					response := makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response.Body).To(Equal(`{}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeUnbindingRequest(instanceID, uniqueBindingID())

					Expect(lastLogLine().Message).To(ContainSubstring("bind.instance-missing"))
					Expect(lastLogLine().Data["error"]).To(ContainSubstring("instance does not exist"))
				})
			})
		})
	})
})
