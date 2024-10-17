// Copyright (C) 2015-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brokerapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi/v11"
	"github.com/pivotal-cf/brokerapi/v11/fakes"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

var _ = Describe("Service Broker API", func() {
	var (
		fakeServiceBroker *fakes.FakeServiceBroker
		brokerAPI         http.Handler
		logBuffer         *gbytes.Buffer
		brokerLogger      *slog.Logger
		apiVersion        string
		credentials       brokerapi.BrokerCredentials
	)

	const requestIdentity = "Request Identity Name"

	makeInstanceProvisioningRequest := func(instanceID string, details map[string]any, queryString string) (response *http.Response) {
		withServer(brokerAPI, func(r requester) {
			path := "/v2/service_instances/" + instanceID + queryString

			buffer := &bytes.Buffer{}
			json.NewEncoder(buffer).Encode(details)
			request, err := http.NewRequest("PUT", path, buffer)
			Expect(err).NotTo(HaveOccurred())
			request.Header.Add("Content-Type", "application/json")
			request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
			if apiVersion != "" {
				request.Header.Add("X-Broker-API-Version", apiVersion)
			}
			request.SetBasicAuth(credentials.Username, credentials.Password)
			response = r.Do(request)
		})
		return response
	}

	makeInstanceProvisioningRequestWithAcceptsIncomplete := func(instanceID string, details map[string]any, acceptsIncomplete bool) *http.Response {
		var acceptsIncompleteFlag string

		if acceptsIncomplete {
			acceptsIncompleteFlag = "?accepts_incomplete=true"
		} else {
			acceptsIncompleteFlag = "?accepts_incomplete=false"
		}

		return makeInstanceProvisioningRequest(instanceID, details, acceptsIncompleteFlag)
	}

	lastLogLine := func() (result map[string]any) {
		GinkgoHelper()

		lines := strings.Split(strings.TrimSpace(string(logBuffer.Contents())), "\n")
		noOfLogLines := len(lines)
		if noOfLogLines == 0 {
			Fail("expected some log lines but there were none")
		}

		Expect(json.Unmarshal([]byte(lines[noOfLogLines-1]), &result)).To(Succeed(), fmt.Sprintf("failed to parse JSON: %q", lines[noOfLogLines-1]))
		return
	}

	BeforeEach(func() {
		apiVersion = "2.14"
		credentials = brokerapi.BrokerCredentials{
			Username: "username",
			Password: "password",
		}
		fakeServiceBroker = &fakes.FakeServiceBroker{
			ProvisionedInstances: map[string]brokerapi.ProvisionDetails{},
			BoundBindings:        map[string]brokerapi.BindDetails{},
			InstanceLimit:        3,
			ServiceID:            "0A789746-596F-4CEA-BFAC-A0795DA056E3",
			PlanID:               "plan-id",
		}

		logBuffer = gbytes.NewBuffer()
		brokerLogger = slog.New(slog.NewJSONHandler(logBuffer, nil))
		brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)
	})

	Describe("response headers", func() {
		makeRequest := func() *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			request := must(http.NewRequest("GET", "/v2/catalog", nil))
			request.SetBasicAuth(credentials.Username, credentials.Password)
			brokerAPI.ServeHTTP(recorder, request)
			return recorder
		}

		It("has a Content-Type header", func() {
			response := makeRequest()

			header := response.Header().Get("Content-Type")
			Expect(header).Should(Equal("application/json"))
		})
	})

	Describe("request context", func() {
		var (
			ctx     context.Context
			reqBody string
		)

		makeRequest := func(method, path, body string) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			request := must(http.NewRequest(method, path, strings.NewReader(body)))
			request.Header.Add("X-Broker-API-Version", "2.14")
			request.SetBasicAuth(credentials.Username, credentials.Password)
			request = request.WithContext(ctx)
			brokerAPI.ServeHTTP(recorder, request)
			return recorder
		}

		BeforeEach(func() {
			ctx = context.WithValue(context.Background(), fakes.FakeBrokerContextDataKey, true)
			reqBody = fmt.Sprintf(`{"service_id":"%s","plan_id":"456"}`, fakeServiceBroker.ServiceID)
		})

		Specify("a catalog endpoint which passes the request context to the broker", func() {
			makeRequest("GET", "/v2/catalog", "")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a provision endpoint which passes the request context to the broker", func() {
			makeRequest("PUT", "/v2/service_instances/instance-id", reqBody)
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a deprovision endpoint which passes the request context to the broker", func() {
			makeRequest("DELETE", "/v2/service_instances/instance-id?service_id=asdf&plan_id=fdsa", "")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a bind endpoint which passes the request context to the broker", func() {
			makeRequest("PUT", "/v2/service_instances/instance-id/service_bindings/binding-id", reqBody)
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("an unbind endpoint which passes the request context to the broker", func() {
			makeRequest("DELETE", "/v2/service_instances/instance-id/service_bindings/binding-id?plan_id=plan-id&service_id=service-id", "")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("an update endpoint which passes the request context to the broker", func() {
			makeRequest("PATCH", "/v2/service_instances/instance-id", `{"service_id":"123"}`)
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a get binding operation endpoint which passes the request context to the broker", func() {
			makeRequest("GET", "/v2/service_instances/instance-id/service_bindings/binding-id", "{}")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a last binding operation endpoint which passes the request context to the broker", func() {
			makeRequest("GET", "/v2/service_instances/instance-id/service_bindings/binding-id/last_operation", "{}")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a last operation endpoint which passes the request context to the broker", func() {
			makeRequest("GET", "/v2/service_instances/instance-id/last_operation", "{}")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})

		Specify("a get instance operation endpoint which passes the request context to the broker", func() {
			makeRequest("GET", "/v2/service_instances/instance-id", "{}")
			Expect(fakeServiceBroker.ReceivedContext).To(BeTrue())
		})
	})

	Describe("authentication", func() {
		makeRequestWithoutAuth := func() (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				request := must(http.NewRequest("GET", "/v2/catalog", nil))
				response = r.Do(request)
			})
			return response
		}

		makeRequestWithUnrecognizedAuth := func() (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				request := must(http.NewRequest("GET", "/v2/catalog", nil))
				// dXNlcm5hbWU6cGFzc3dvcmQ= is base64 encoding of 'username:password',
				// ie, a correctly encoded basic authorization header
				request.Header["Authorization"] = []string{"NOTBASIC dXNlcm5hbWU6cGFzc3dvcmQ="}

				response = r.Do(request)
			})
			return response
		}

		When("using default basic authentication", func() {
			makeRequestWithBasicAuth := func(username string, password string) (response *http.Response) {
				withServer(brokerAPI, func(r requester) {
					request := must(http.NewRequest("GET", "/v2/catalog", nil))
					request.SetBasicAuth(username, password)
					request.Header.Add("Content-Type", "application/json")
					request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
					request.Header.Add("X-Broker-API-Version", apiVersion)

					response = r.Do(request)
				})
				return response
			}

			It("returns 401 when the authorization header has an incorrect password", func() {
				response := makeRequestWithBasicAuth("username", "fake_password")
				Expect(response).To(HaveHTTPStatus(http.StatusUnauthorized))
			})

			It("returns 401 when the authorization header has an incorrect username", func() {
				response := makeRequestWithBasicAuth("fake_username", "password")
				Expect(response).To(HaveHTTPStatus(http.StatusUnauthorized))
			})

			It("returns 401 when there is no authorization header", func() {
				response := makeRequestWithoutAuth()
				Expect(response).To(HaveHTTPStatus(http.StatusUnauthorized))
			})

			It("returns 401 when there is an unrecognized authorization header", func() {
				response := makeRequestWithUnrecognizedAuth()
				Expect(response).To(HaveHTTPStatus(http.StatusUnauthorized))
			})

			It("does not call through to the service broker when not authenticated", func() {
				makeRequestWithBasicAuth("username", "fake_password")
				Ω(fakeServiceBroker.BrokerCalled).ShouldNot(BeTrue(),
					"broker should not have been hit when authentication failed",
				)
			})

			It("calls through to the service broker when authenticated", func() {
				makeRequestWithBasicAuth(credentials.Username, credentials.Password)
				Ω(fakeServiceBroker.BrokerCalled).Should(BeTrue(),
					"broker should have been hit when authentication succeeded",
				)
			})
		})

		When("using custom authentication", func() {
			expectedToken := "expected_token"

			makeRequestWithBearerTokenAuth := func(token string) (response *http.Response) {
				withServer(brokerAPI, func(r requester) {
					request := must(http.NewRequest("GET", "/v2/catalog", nil))
					request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
					request.Header.Add("Content-Type", "application/json")
					request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
					request.Header.Add("X-Broker-API-Version", apiVersion)

					response = r.Do(request)
				})
				return response
			}

			authMiddleware := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					unauthorized := func(w http.ResponseWriter) {
						http.Error(w, "Not Authorized", http.StatusUnauthorized)
					}
					badRequest := func(w http.ResponseWriter) {
						http.Error(w, "Unable to determine Authorization method, supported are 'Basic' and 'Bearer'.", http.StatusBadRequest)
					}

					authHeader := r.Header.Get("Authorization")
					if authHeader == "" {
						unauthorized(w)
						return
					}

					authHeaderParts := strings.Fields(authHeader)
					if len(authHeaderParts) < 2 {
						badRequest(w)
						return
					}

					authMethod := authHeaderParts[0]
					if authMethod != "Bearer" {
						unauthorized(w)
						return
					}

					authToken := strings.Join(authHeaderParts[1:], " ")
					if authToken != expectedToken {
						unauthorized(w)
						return
					}

					next.ServeHTTP(w, r)
				})
			}

			BeforeEach(func() {
				brokerAPI = brokerapi.NewWithCustomAuth(fakeServiceBroker, brokerLogger, authMiddleware)
			})

			It("returns 401 when the authorization header has an incorrect bearer token", func() {
				response := makeRequestWithBearerTokenAuth("incorrect_token")
				Expect(response).To(HaveHTTPStatus(http.StatusUnauthorized))
			})

			It("does not call through to the service broker when not authenticated", func() {
				makeRequestWithBearerTokenAuth("incorrect_token")
				Ω(fakeServiceBroker.BrokerCalled).ShouldNot(BeTrue(),
					"broker should not have been hit when authentication failed",
				)
			})

			It("calls through to the service broker when authenticated", func() {
				makeRequestWithBearerTokenAuth("expected_token")
				Ω(fakeServiceBroker.BrokerCalled).Should(BeTrue(),
					"broker should have been hit when authentication succeeds",
				)
			})
		})
	})

	Describe("OriginatingIdentityHeader", func() {

		var (
			fakeServiceBroker *fakes.AutoFakeServiceBroker
			req               *http.Request
			testServer        *httptest.Server
		)

		BeforeEach(func() {
			fakeServiceBroker = new(fakes.AutoFakeServiceBroker)
			brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)

			testServer = httptest.NewServer(brokerAPI)
			var err error
			req, err = http.NewRequest("GET", testServer.URL+"/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Broker-API-Version", "2.14")
			req.SetBasicAuth(credentials.Username, credentials.Password)
		})

		AfterEach(func() {
			testServer.Close()
		})

		When("X-Broker-API-Originating-Identity is passed", func() {
			It("Adds it to the context", func() {
				originatingIdentity := "Originating Identity Name"
				req.Header.Add("X-Broker-API-Originating-Identity", originatingIdentity)

				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.OriginatingIdentityKey)).To(Equal(originatingIdentity))
			})
		})

		When("X-Broker-API-Originating-Identity is not passed", func() {
			It("Adds empty originatingIdentity to the context", func() {
				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.OriginatingIdentityKey)).To(Equal(""))
			})
		})
	})

	Describe("RequestIdentityHeader", func() {
		var (
			fakeServiceBroker *fakes.AutoFakeServiceBroker
			req               *http.Request
			testServer        *httptest.Server
		)

		BeforeEach(func() {
			fakeServiceBroker = new(fakes.AutoFakeServiceBroker)
			brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)

			testServer = httptest.NewServer(brokerAPI)
			var err error
			req, err = http.NewRequest("GET", testServer.URL+"/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Broker-API-Version", "2.14")
			req.SetBasicAuth(credentials.Username, credentials.Password)
		})

		AfterEach(func() {
			testServer.Close()
		})

		When("X-Broker-API-Request-Identity is passed", func() {
			It("adds it to the context and returns in response", func() {
				const requestIdentity = "Request Identity Name"
				req.Header.Add("X-Broker-API-Request-Identity", requestIdentity)

				response, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.RequestIdentityKey)).To(Equal(requestIdentity))

				header := response.Header.Get("X-Broker-API-Request-Identity")
				Expect(header).To(Equal(requestIdentity))
			})
		})

		When("X-Broker-API-Request-Identity is not passed", func() {
			It("adds empty requestIdentity to the context and does not return in response", func() {
				response, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.RequestIdentityKey)).To(Equal(""))

				header := response.Header.Get("X-Broker-API-Request-Identity")
				Expect(header).To(Equal(""))
			})
		})
	})

	Describe("InfoLocationHeader", func() {

		var (
			fakeServiceBroker *fakes.AutoFakeServiceBroker
			req               *http.Request
			testServer        *httptest.Server
		)

		BeforeEach(func() {
			fakeServiceBroker = new(fakes.AutoFakeServiceBroker)
			brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)

			testServer = httptest.NewServer(brokerAPI)
			var err error
			req, err = http.NewRequest("GET", testServer.URL+"/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Broker-API-Version", "2.14")
			req.SetBasicAuth(credentials.Username, credentials.Password)
		})

		AfterEach(func() {
			testServer.Close()
		})

		When("X-Api-Info-Location is passed", func() {
			It("Adds it to the context", func() {
				infoLocation := "API Info Location Value"
				req.Header.Add("X-Api-Info-Location", infoLocation)

				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.InfoLocationKey)).To(Equal(infoLocation))

			})
		})

		When("X-Api-Info-Location is not passed", func() {
			It("Adds empty infoLocation to the context", func() {
				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.InfoLocationKey)).To(Equal(""))
			})
		})
	})

	Describe("CorrelationIDHeader", func() {
		const correlationID = "fake-correlation-id"

		type testCase struct {
			correlationIDHeaderName string
		}

		var (
			fakeServiceBroker *fakes.AutoFakeServiceBroker
			req               *http.Request
			testServer        *httptest.Server
		)

		BeforeEach(func() {
			fakeServiceBroker = new(fakes.AutoFakeServiceBroker)
			brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)

			testServer = httptest.NewServer(brokerAPI)
			var err error
			req, err = http.NewRequest("GET", testServer.URL+"/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("X-Broker-API-Version", "2.14")
			req.SetBasicAuth(credentials.Username, credentials.Password)
		})

		AfterEach(func() {
			testServer.Close()
		})

		DescribeTable("Adds correlation id to the context",
			func(tc testCase) {
				req.Header.Add(tc.correlationIDHeaderName, correlationID)

				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.CorrelationIDKey)).To(Equal(correlationID))
			},
			Entry("X-Correlation-ID", testCase{
				correlationIDHeaderName: "X-Correlation-ID",
			}),
			Entry("X-CorrelationID", testCase{
				correlationIDHeaderName: "X-CorrelationID",
			}),
			Entry("X-ForRequest-ID", testCase{
				correlationIDHeaderName: "X-ForRequest-ID",
			}),
			Entry("X-Request-ID", testCase{
				correlationIDHeaderName: "X-Request-ID",
			}),
			Entry("X-Vcap-Request-Id", testCase{
				correlationIDHeaderName: "X-Vcap-Request-Id",
			}),
		)

		When("X-Correlation-ID is not passed", func() {
			It("Generates correlation id and adds it to the context", func() {
				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeServiceBroker.ServicesCallCount()).To(Equal(1), "Services was not called")
				ctx := fakeServiceBroker.ServicesArgsForCall(0)
				Expect(ctx.Value(middlewares.CorrelationIDKey)).To(Not(BeNil()))
			})
		})
	})

	Describe("catalog endpoint", func() {
		const requestIdentity = "Request Identity Name"
		makeCatalogRequest := func(apiVersion string, fail bool) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			request := must(http.NewRequest(http.MethodGet, "/v2/catalog", nil))
			if apiVersion != "" {
				request.Header.Add("X-Broker-API-Version", apiVersion)
			}
			request.SetBasicAuth(credentials.Username, credentials.Password)
			request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
			ctx := context.Background()
			if fail {
				ctx = context.WithValue(ctx, fakes.FakeBrokerContextFailsKey, true)
			}
			request = request.WithContext(ctx)
			brokerAPI.ServeHTTP(recorder, request)
			return recorder
		}

		It("returns a 200", func() {
			response := makeCatalogRequest("2.14", false)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Header().Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
			Expect(response.Body).To(MatchJSON(fixture("catalog.json")))
		})

		It("returns a 500", func() {
			response := makeCatalogRequest("2.14", true)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Header().Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
			Expect(response.Body.String()).To(MatchJSON(`{ "description": "something went wrong!" }`))
		})

		Context("the request is malformed", func() {
			It("missing header X-Broker-API-Version", func() {
				response := makeCatalogRequest("", false)

				Expect(response.Code).To(Equal(http.StatusPreconditionFailed))
				Expect(response.Header().Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
			})

			It("has wrong version of API", func() {
				response := makeCatalogRequest("1.14", false)

				Expect(response.Code).To(Equal(http.StatusPreconditionFailed))
				Expect(response.Header().Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
			})
		})
	})

	Describe("instance lifecycle endpoint", func() {
		makeGetInstanceWithQueryParamsRequest := func(instanceID string, params map[string]string) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s", instanceID)
				request, err := http.NewRequest("GET", path, strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				request.Header.Add("X-Broker-API-Version", apiVersion)
				request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
				request.SetBasicAuth("username", "password")
				q := request.URL.Query()
				for query, value := range params {
					q.Add(query, value)
				}
				request.URL.RawQuery = q.Encode()
				response = r.Do(request)
			})
			return response
		}

		makeGetInstanceRequest := func(instanceID string) *http.Response {
			return makeGetInstanceWithQueryParamsRequest(instanceID, map[string]string{})
		}

		makeInstanceDeprovisioningRequestFull := func(instanceID, serviceID, planID, queryString string) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s?plan_id=%s&service_id=%s", instanceID, planID, serviceID)
				if queryString != "" {
					path = fmt.Sprintf("%s&%s", path, queryString)
				}
				request, err := http.NewRequest("DELETE", path, strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")
				request.Header.Add("X-Broker-API-Version", apiVersion)
				request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
				response = r.Do(request)
			})
			return response
		}

		makeInstanceDeprovisioningRequest := func(instanceID, queryString string) *http.Response {
			return makeInstanceDeprovisioningRequestFull(instanceID, "service-id", "plan-id", queryString)
		}

		Describe("provisioning", func() {
			var instanceID string
			var provisionDetails map[string]any

			BeforeEach(func() {
				instanceID = uniqueInstanceID()
				provisionDetails = map[string]any{
					"service_id":        fakeServiceBroker.ServiceID,
					"plan_id":           "plan-id",
					"organization_guid": "organization-guid",
					"space_guid":        "space-guid",
					"maintenance_info": map[string]any{
						"public": map[string]string{
							"k8s-version": "0.0.1-alpha2",
						},
						"private": "just a sha thing",
					},
				}
			})

			It("calls Provision on the service broker with all params", func() {
				makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				Expect(fakeServiceBroker.ProvisionedInstances[instanceID]).To(Equal(brokerapi.ProvisionDetails{
					ServiceID:        fakeServiceBroker.ServiceID,
					PlanID:           "plan-id",
					OrganizationGUID: "organization-guid",
					SpaceGUID:        "space-guid",
					MaintenanceInfo: &brokerapi.MaintenanceInfo{
						Public: map[string]string{
							"k8s-version": "0.0.1-alpha2",
						},
						Private: "just a sha thing",
					},
				}))
			})

			It("calls Provision on the service broker with the instance id", func() {
				makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				_, ok := fakeServiceBroker.ProvisionedInstances[instanceID]
				Expect(ok).To(BeTrue())
			})

			It("calls GetInstance on the service broker with the instance id", func() {
				makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				_, ok := fakeServiceBroker.ProvisionedInstances[instanceID]
				Expect(ok).To(BeTrue())
				fakeServiceBroker.DashboardURL = "https://example.com/dashboard/some-instance"
				resp := makeGetInstanceRequest(instanceID)
				Expect(fakeServiceBroker.GetInstanceIDs).To(ContainElement(instanceID))
				Expect(readBody(resp)).To(MatchJSON(fixture("get_instance.json")))
				Expect(resp.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
			})

			Context("when the broker returns some operation data", func() {
				BeforeEach(func() {
					fakeServiceBroker = &fakes.FakeServiceBroker{
						ProvisionedInstances:  map[string]brokerapi.ProvisionDetails{},
						BoundBindings:         map[string]brokerapi.BindDetails{},
						InstanceLimit:         3,
						OperationDataToReturn: "some-operation-data",
						ServiceID:             fakeServiceBroker.ServiceID,
						PlanID:                fakeServiceBroker.PlanID,
					}
					fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
						FakeServiceBroker:    *fakeServiceBroker,
						ShouldProvisionAsync: true,
					}
					brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
				})

				It("returns the operation data to the cloud controller", func() {
					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(readBody(response)).To(MatchJSON(fixture("operation_data_response.json")))
				})
			})

			Context("when there are arbitrary params", func() {
				var rawParams string
				var rawCtx string

				BeforeEach(func() {
					provisionDetails["parameters"] = map[string]any{
						"string": "some-string",
						"number": 1,
						"object": struct{ Name string }{"some-name"},
						"array":  []any{"a", "b", "c"},
					}
					rawParams = `{
					"string":"some-string",
					"number":1,
					"object": { "Name": "some-name" },
					"array": [ "a", "b", "c" ]
				}`
					provisionDetails["context"] = map[string]any{
						"platform":      "fake-platform",
						"serial-number": 12648430,
						"object":        struct{ Name string }{"parameter"},
						"array":         []any{"1", "2", "3"},
					}
					rawCtx = `{
					"platform":"fake-platform",
					"serial-number":12648430,
					"object": {"Name":"parameter"},
					"array":[ "1", "2", "3" ]
				}`
				})

				It("calls Provision on the service broker with all params", func() {
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(string(fakeServiceBroker.ProvisionedInstances[instanceID].RawParameters)).To(MatchJSON(rawParams))
				})

				It("calls Provision with details with raw parameters", func() {
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					detailsWithRawParameters := brokerapi.DetailsWithRawParameters(fakeServiceBroker.ProvisionedInstances[instanceID])
					rawParameters := detailsWithRawParameters.GetRawParameters()
					Expect(string(rawParameters)).To(MatchJSON(rawParams))
				})

				It("calls Provision with details with raw context", func() {
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					detailsWithRawContext := brokerapi.DetailsWithRawContext(fakeServiceBroker.ProvisionedInstances[instanceID])
					rawContext := detailsWithRawContext.GetRawContext()
					Expect(string(rawContext)).To(MatchJSON(rawCtx))
				})
			})

			Context("when the instance does not exist", func() {
				It("returns a 201 with empty JSON", func() {
					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(response).To(HaveHTTPStatus(http.StatusCreated))
					Expect(readBody(response)).To(MatchJSON(fixture("provisioning.json")))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				})

				Context("when the broker returns a dashboard URL", func() {
					BeforeEach(func() {
						fakeServiceBroker.DashboardURL = "some-dashboard-url"
					})

					It("returns json with dashboard URL", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(readBody(response)).To(MatchJSON(fixture("provisioning_with_dashboard.json")))
					})
				})

				Context("when the instance limit has been reached", func() {
					BeforeEach(func() {
						for i := 0; i < fakeServiceBroker.InstanceLimit; i++ {
							makeInstanceProvisioningRequest(uniqueInstanceID(), provisionDetails, "")
						}
					})

					It("returns a 500 with error", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

						Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
						Expect(readBody(response)).To(MatchJSON(fixture("instance_limit_error.json")))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.instance-limit-reached"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance limit for this service has been reached"))
					})
				})

				Context("when an unexpected error occurs", func() {
					BeforeEach(func() {
						fakeServiceBroker.ProvisionError = errors.New("broker failed")
					})

					It("returns a 500 with error", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
						Expect(readBody(response)).To(MatchJSON(`{"description":"broker failed"}`))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.unknown-error"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "broker failed"))
					})
				})

				Context("when a custom error occurs", func() {
					BeforeEach(func() {
						fakeServiceBroker.ProvisionError = brokerapi.NewFailureResponse(
							errors.New("I failed in unique and interesting ways"),
							http.StatusTeapot,
							"interesting-failure",
						)
					})

					It("returns status teapot with error", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
						Expect(readBody(response)).To(MatchJSON(`{"description":"I failed in unique and interesting ways"}`))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.interesting-failure"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "I failed in unique and interesting ways"))
					})
				})

				Context("RawParameters are not valid JSON", func() {
					BeforeEach(func() {
						fakeServiceBroker.ProvisionError = brokerapi.ErrRawParamsInvalid
					})

					It("returns a 422", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{"description":"The format of the parameters is not valid JSON"}`))
					})

					It("logs an appropriate error", func() {
						makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.invalid-raw-params"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "The format of the parameters is not valid JSON"))
					})
				})

				Context("when we send invalid json", func() {
					makeBadInstanceProvisioningRequest := func(instanceID string) (response *http.Response) {
						withServer(brokerAPI, func(r requester) {
							path := "/v2/service_instances/" + instanceID

							body := strings.NewReader("{{{{{")
							request, err := http.NewRequest("PUT", path, body)
							Expect(err).NotTo(HaveOccurred())
							request.Header.Add("Content-Type", "application/json")
							request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
							if apiVersion != "" {
								request.Header.Add("X-Broker-Api-Version", apiVersion)
							}
							request.SetBasicAuth(credentials.Username, credentials.Password)
							response = r.Do(request)
						})

						return response
					}

					It("returns a 422 bad request", func() {
						response := makeBadInstanceProvisioningRequest(instanceID)
						Expect(response.StatusCode).Should(Equal(http.StatusUnprocessableEntity))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})

					It("logs a message", func() {
						makeBadInstanceProvisioningRequest(instanceID)
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.invalid-service-details"))
					})
				})
			})

			Context("when the instance already exists", func() {
				BeforeEach(func() {
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				})

				Context("returns a StatusOK on", func() {
					It("sync broker response", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})

					It("async broker response", func() {
						fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
							FakeServiceBroker:    *fakeServiceBroker,
							ShouldProvisionAsync: true,
						}
						brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))

						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})
				})

				It("returns a StatusConflict", func() {
					By("Service Instance with the same id already exists and is being provisioned but with different attributes")
					provisionDetails["space_guid"] = "fake-space_guid"
					defer func() {
						By("Return default value")
						provisionDetails["space_guid"] = "space_guid"
					}()
					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(response).To(HaveHTTPStatus(http.StatusConflict))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				})

				It("returns an empty JSON object", func() {
					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(readBody(response)).To(MatchJSON(`{}`))
				})

				It("logs an appropriate error when respond with statusConflict", func() {
					By("Service Instance with the same id already exists and is being provisioned but with different attributes")
					provisionDetails["space_guid"] = "fake-space_guid"
					defer func() {
						By("Return default value")
						provisionDetails["space_guid"] = "space_guid"
					}()
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.instance-already-exists"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance already exists"))
				})
			})

			Describe("accepts_incomplete", func() {
				Context("when the accepts_incomplete flag is true", func() {
					It("calls ProvisionAsync on the service broker", func() {
						acceptsIncomplete := true
						makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, acceptsIncomplete)
						Expect(fakeServiceBroker.ProvisionedInstances[instanceID]).To(Equal(brokerapi.ProvisionDetails{
							ServiceID:        fakeServiceBroker.ServiceID,
							PlanID:           "plan-id",
							OrganizationGUID: "organization-guid",
							SpaceGUID:        "space-guid",
							MaintenanceInfo: &brokerapi.MaintenanceInfo{
								Public: map[string]string{
									"k8s-version": "0.0.1-alpha2",
								},
								Private: "just a sha thing",
							},
						}))

						_, ok := fakeServiceBroker.ProvisionedInstances[instanceID]
						Expect(ok).To(BeTrue())
					})

					Context("when the broker chooses to provision asynchronously", func() {
						BeforeEach(func() {
							fakeServiceBroker = &fakes.FakeServiceBroker{
								ProvisionedInstances: map[string]brokerapi.ProvisionDetails{},
								BoundBindings:        map[string]brokerapi.BindDetails{},
								InstanceLimit:        3,
								ServiceID:            fakeServiceBroker.ServiceID,
								PlanID:               fakeServiceBroker.PlanID,
							}
							fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
								FakeServiceBroker:    *fakeServiceBroker,
								ShouldProvisionAsync: true,
							}
							brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
						})

						It("returns a 202", func() {
							response := makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, true)
							Expect(response).To(HaveHTTPStatus(http.StatusAccepted))
						})
					})

					Context("when the broker chooses to provision synchronously", func() {
						BeforeEach(func() {
							fakeServiceBroker = &fakes.FakeServiceBroker{
								ProvisionedInstances: map[string]brokerapi.ProvisionDetails{},
								BoundBindings:        map[string]brokerapi.BindDetails{},
								InstanceLimit:        3,
								ServiceID:            fakeServiceBroker.ServiceID,
								PlanID:               fakeServiceBroker.PlanID,
							}
							fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
								FakeServiceBroker:    *fakeServiceBroker,
								ShouldProvisionAsync: false,
							}
							brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
						})

						It("returns a 201", func() {
							response := makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, true)
							Expect(response).To(HaveHTTPStatus(http.StatusCreated))
						})
					})
				})

				Context("when the accepts_incomplete flag is false", func() {
					It("returns a 201", func() {
						response := makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, false)
						Expect(response).To(HaveHTTPStatus(http.StatusCreated))
					})

					Context("when broker can only respond asynchronously", func() {
						BeforeEach(func() {
							fakeServiceBroker = &fakes.FakeServiceBroker{
								ProvisionedInstances: map[string]brokerapi.ProvisionDetails{},
								BoundBindings:        map[string]brokerapi.BindDetails{},
								InstanceLimit:        3,
								ServiceID:            fakeServiceBroker.ServiceID,
								PlanID:               fakeServiceBroker.PlanID,
							}
							fakeAsyncServiceBroker := &fakes.FakeAsyncOnlyServiceBroker{
								FakeServiceBroker: *fakeServiceBroker,
							}
							brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
						})

						It("returns a 422", func() {
							acceptsIncomplete := false
							response := makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, acceptsIncomplete)
							Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
							Expect(readBody(response)).To(MatchJSON(fixture("async_required.json")))
						})
					})
				})

				Context("when the accepts_incomplete flag is missing", func() {
					It("returns a 201", func() {
						response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
						Expect(response).To(HaveHTTPStatus(http.StatusCreated))
					})

					Context("when broker can only respond asynchronously", func() {
						BeforeEach(func() {
							fakeServiceBroker = &fakes.FakeServiceBroker{
								ProvisionedInstances: map[string]brokerapi.ProvisionDetails{},
								BoundBindings:        map[string]brokerapi.BindDetails{},
								InstanceLimit:        3,
								ServiceID:            fakeServiceBroker.ServiceID,
								PlanID:               fakeServiceBroker.PlanID,
							}
							fakeAsyncServiceBroker := &fakes.FakeAsyncOnlyServiceBroker{
								FakeServiceBroker: *fakeServiceBroker,
							}
							brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
						})

						It("returns a 422", func() {
							acceptsIncomplete := false
							response := makeInstanceProvisioningRequestWithAcceptsIncomplete(instanceID, provisionDetails, acceptsIncomplete)
							Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
							Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
							Expect(readBody(response)).To(MatchJSON(fixture("async_required.json")))
						})
					})
				})
			})

			Context("the request is malformed", func() {
				It("missing header X-Broker-API-Version", func() {
					apiVersion = ""

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					apiVersion = "1.14"

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})

				It("missing service_id", func() {
					delete(provisionDetails, "service_id")

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.service-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "service_id missing"))
				})

				It("missing plan_id", func() {
					delete(provisionDetails, "plan_id")

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.plan-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "plan_id missing"))
				})

				It("service_id not in the catalog", func() {
					provisionDetails["service_id"] = "not-in-the-catalogue"

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.invalid-service-id"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "service-id not in the catalog"))
				})

				It("plan_id not in the catalog", func() {
					provisionDetails["plan_id"] = "not-in-the-catalogue"

					response := makeInstanceProvisioningRequest(instanceID, provisionDetails, "")

					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "provision.invalid-plan-id"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "plan-id not in the catalog"))
				})
			})
		})

		Describe("updating", func() {
			var (
				instanceID  string
				details     map[string]any
				queryString string
				response    *http.Response
			)
			const updateRequestIdentity = "Update Request Identity Name"

			makeInstanceUpdateRequest := func(instanceID string, details map[string]any, queryString string, apiVersion string) (response *http.Response) {
				withServer(brokerAPI, func(r requester) {
					path := "/v2/service_instances/" + instanceID + queryString

					buffer := &bytes.Buffer{}
					json.NewEncoder(buffer).Encode(details)
					request, err := http.NewRequest("PATCH", path, buffer)
					Expect(err).NotTo(HaveOccurred())
					if apiVersion != "" {
						request.Header.Add("X-Broker-Api-Version", apiVersion)
					}
					request.Header.Add("Content-Type", "application/json")
					request.SetBasicAuth(credentials.Username, credentials.Password)
					request.Header.Add("X-Broker-API-Request-Identity", updateRequestIdentity)

					response = r.Do(request)
				})
				return response
			}

			BeforeEach(func() {
				instanceID = uniqueInstanceID()
				details = map[string]any{
					"service_id": "some-service-id",
					"plan_id":    "new-plan",
					"parameters": map[string]any{
						"new-param": "new-param-value",
					},
					"previous_values": map[string]any{
						"service_id":      "service-id",
						"plan_id":         "old-plan",
						"organization_id": "org-id",
						"space_id":        "space-id",
					},
					"context": map[string]any{
						"new-context": "new-context-value",
					},
					"maintenance_info": map[string]any{
						"public": map[string]string{
							"k8s-version": "0.0.1-alpha2",
						},
						"private": "just a sha thing",
					},
				}
				queryString = "?accept_incomplete=true"
			})

			JustBeforeEach(func() {
				response = makeInstanceUpdateRequest(instanceID, details, queryString, "2.14")
			})

			Context("the request is malformed", func() {
				It("missing header X-Broker-API-Version", func() {
					instanceID := "instance-id"
					response := makeInstanceUpdateRequest(instanceID, details, queryString, "")

					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					instanceID := "instance-id"
					response := makeInstanceUpdateRequest(instanceID, details, queryString, "1.14")

					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})

				It("missing service-id", func() {
					delete(details, "service_id")

					response := makeInstanceUpdateRequest("instance-id", details, queryString, "2.14")

					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "update.service-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "service_id missing"))
				})
			})

			Context("when the broker returns no error", func() {
				Context("when the broker responds synchronously", func() {
					It("returns HTTP 200", func() {
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))
						Expect(readBody(response)).To(Equal("{}\n"))
					})

					It("calls broker with instanceID and update details", func() {
						Expect(fakeServiceBroker.UpdatedInstanceIDs).To(ConsistOf(instanceID))
						Expect(fakeServiceBroker.UpdateDetails.ServiceID).To(Equal("some-service-id"))
						Expect(fakeServiceBroker.UpdateDetails.PlanID).To(Equal("new-plan"))
						Expect(fakeServiceBroker.UpdateDetails.PreviousValues).To(Equal(brokerapi.PreviousValues{
							PlanID:    "old-plan",
							ServiceID: "service-id",
							OrgID:     "org-id",
							SpaceID:   "space-id",
						},
						))
						Expect(fakeServiceBroker.UpdateDetails.RawParameters).To(Equal(json.RawMessage(`{"new-param":"new-param-value"}`)))
						Expect(*fakeServiceBroker.UpdateDetails.MaintenanceInfo).To(Equal(brokerapi.MaintenanceInfo{
							Public:  map[string]string{"k8s-version": "0.0.1-alpha2"},
							Private: "just a sha thing"},
						))
					})

					It("calls update with details with raw parameters", func() {
						detailsWithRawParameters := brokerapi.DetailsWithRawParameters(fakeServiceBroker.UpdateDetails)
						rawParameters := detailsWithRawParameters.GetRawParameters()
						Expect(rawParameters).To(Equal(json.RawMessage(`{"new-param":"new-param-value"}`)))
					})

					It("calls update with details with raw context", func() {
						detailsWithRawContext := brokerapi.DetailsWithRawContext(fakeServiceBroker.UpdateDetails)
						rawContext := detailsWithRawContext.GetRawContext()
						Expect(string(rawContext)).To(
							MatchJSON(`{"new-context":"new-context-value"}`),
						)
					})

					Context("when accepts_incomplete=true", func() {
						BeforeEach(func() {
							queryString = "?accepts_incomplete=true"
						})

						It("tells broker async is allowed", func() {
							Expect(fakeServiceBroker.AsyncAllowed).To(BeTrue())
						})
					})

					Context("when accepts_incomplete is not supplied", func() {
						BeforeEach(func() {
							queryString = ""
						})

						It("tells broker async not allowed", func() {
							Expect(fakeServiceBroker.AsyncAllowed).To(BeFalse())
						})
					})
				})

				Context("when the broker responds asynchronously", func() {
					BeforeEach(func() {
						fakeServiceBroker.ShouldReturnAsync = true
					})

					It("returns HTTP 202", func() {
						Expect(response).To(HaveHTTPStatus(http.StatusAccepted))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))
					})

					Context("when the broker responds with operation data", func() {
						BeforeEach(func() {
							fakeServiceBroker.OperationDataToReturn = "some-operation-data"
						})

						It("returns the operation data to the cloud controller", func() {
							Expect(readBody(response)).To(MatchJSON(fixture("operation_data_response.json")))
						})
					})
				})

				Context("when the broker returns a dashboard URL", func() {
					BeforeEach(func() {
						fakeServiceBroker.DashboardURL = "some-dashboard-url"
					})

					It("returns json with dashboard URL", func() {
						response := makeInstanceUpdateRequest(instanceID, details, "", "2.14")
						Expect(readBody(response)).To(MatchJSON(fixture("updating_with_dashboard.json")))
					})
				})

			})

			Context("when the broker indicates that it needs async support", func() {
				BeforeEach(func() {
					fakeServiceBroker.UpdateError = brokerapi.ErrAsyncRequired
				})

				It("returns HTTP 422 with error", func() {
					Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))

					body := unmarshalBody(response)
					Expect(body["error"]).To(Equal("AsyncRequired"))
					Expect(body["description"]).To(Equal("This service plan requires client support for asynchronous service operations."))
				})
			})

			Context("when the broker indicates that the plan cannot be upgraded", func() {
				BeforeEach(func() {
					fakeServiceBroker.UpdateError = brokerapi.ErrPlanChangeNotSupported
				})

				It("returns HTTP 422 with error", func() {
					Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))

					body := unmarshalBody(response)
					Expect(body["error"]).To(Equal("PlanChangeNotSupported"))
					Expect(body["description"]).To(Equal("The requested plan migration cannot be performed"))
				})
			})

			Context("when the broker errors in an unknown way", func() {
				BeforeEach(func() {
					fakeServiceBroker.UpdateError = errors.New("some horrible internal error")
				})

				It("returns HTTP 500", func() {
					Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(updateRequestIdentity))

					body := unmarshalBody(response)
					Expect(body["description"]).To(Equal("some horrible internal error"))
				})
			})
		})

		Describe("deprovisioning", func() {
			It("calls Deprovision on the service broker with the instance id", func() {
				instanceID := uniqueInstanceID()
				makeInstanceDeprovisioningRequest(instanceID, "")
				Expect(fakeServiceBroker.DeprovisionedInstanceIDs).To(ContainElement(instanceID))
			})

			Context("when the instance exists", func() {
				var instanceID string
				var provisionDetails map[string]any

				BeforeEach(func() {
					instanceID = uniqueInstanceID()

					provisionDetails = map[string]any{
						"service_id":        fakeServiceBroker.ServiceID,
						"plan_id":           "plan-id",
						"organization_guid": "organization-guid",
						"space_guid":        "space-guid",
					}
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				})

				itReturnsStatus := func(expectedStatus int, queryString string) {
					It(fmt.Sprintf("returns HTTP %d", expectedStatus), func() {
						response := makeInstanceDeprovisioningRequest(instanceID, queryString)
						Expect(response.StatusCode).To(Equal(expectedStatus))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					})
				}

				itReturnsEmptyJsonObject := func(queryString string) {
					It("returns an empty JSON object", func() {
						response := makeInstanceDeprovisioningRequest(instanceID, queryString)
						Expect(readBody(response)).To(MatchJSON(`{}`))
					})
				}

				Context("when the broker can only operate synchronously", func() {
					Context("when the accepts_incomplete flag is not set", func() {
						itReturnsStatus(http.StatusOK, "")
						itReturnsEmptyJsonObject("")
					})

					Context("when the accepts_incomplete flag is set to true", func() {
						itReturnsStatus(http.StatusOK, "accepts_incomplete=true")
						itReturnsEmptyJsonObject("accepts_incomplete=true")
					})
				})

				Context("when the broker can only operate asynchronously", func() {
					BeforeEach(func() {
						fakeAsyncServiceBroker := &fakes.FakeAsyncOnlyServiceBroker{
							FakeServiceBroker: *fakeServiceBroker,
						}
						brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
					})

					Context("when the accepts_incomplete flag is not set", func() {
						itReturnsStatus(http.StatusUnprocessableEntity, "")

						It("returns a descriptive error", func() {
							response := makeInstanceDeprovisioningRequest(instanceID, "")
							Expect(readBody(response)).To(MatchJSON(fixture("async_required.json")))
						})
					})

					Context("when the accepts_incomplete flag is set to true", func() {
						itReturnsStatus(http.StatusAccepted, "accepts_incomplete=true")
						itReturnsEmptyJsonObject("accepts_incomplete=true")
					})

					Context("when the broker returns operation data", func() {
						BeforeEach(func() {
							fakeServiceBroker.OperationDataToReturn = "some-operation-data"
							fakeAsyncServiceBroker := &fakes.FakeAsyncOnlyServiceBroker{
								FakeServiceBroker: *fakeServiceBroker,
							}
							brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
						})

						itReturnsStatus(http.StatusAccepted, "accepts_incomplete=true")

						It("returns the operation data to the cloud controller", func() {
							response := makeInstanceDeprovisioningRequest(instanceID, "accepts_incomplete=true")
							Expect(readBody(response)).To(MatchJSON(fixture("operation_data_response.json")))
						})
					})
				})

				Context("when the broker can operate both synchronously and asynchronously", func() {
					BeforeEach(func() {
						fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
							FakeServiceBroker: *fakeServiceBroker,
						}
						brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
					})

					Context("when the accepts_incomplete flag is not set", func() {
						itReturnsStatus(http.StatusOK, "")
						itReturnsEmptyJsonObject("")
					})

					Context("when the accepts_incomplete flag is set to true", func() {
						itReturnsStatus(http.StatusAccepted, "accepts_incomplete=true")
						itReturnsEmptyJsonObject("accepts_incomplete=true")
					})
				})

				It("contains plan_id", func() {
					makeInstanceDeprovisioningRequest(instanceID, "")
					Expect(fakeServiceBroker.DeprovisionDetails.PlanID).To(Equal("plan-id"))
				})

				It("contains service_id", func() {
					makeInstanceDeprovisioningRequest(instanceID, "")
					Expect(fakeServiceBroker.DeprovisionDetails.ServiceID).To(Equal("service-id"))
				})

				When("force query param", func() {
					It("contains force as true when true is passed", func() {
						makeInstanceDeprovisioningRequest(instanceID, "force=true")
						Expect(fakeServiceBroker.DeprovisionDetails.Force).To(BeTrue())
					})

					It("contains force as false when false is passed", func() {
						makeInstanceDeprovisioningRequest(instanceID, "force=false")
						Expect(fakeServiceBroker.DeprovisionDetails.Force).To(BeFalse())
					})

					It("contains force as false when it is not passed", func() {
						makeInstanceDeprovisioningRequest(instanceID, "")
						Expect(fakeServiceBroker.DeprovisionDetails.Force).To(BeFalse())
					})
				})
			})

			Context("when the instance does not exist", func() {
				var instanceID string

				It("returns a 410", func() {
					response := makeInstanceDeprovisioningRequest(uniqueInstanceID(), "")

					Expect(response).To(HaveHTTPStatus(http.StatusGone))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeInstanceDeprovisioningRequest(instanceID, "")
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "deprovision.instance-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance does not exist"))
				})
			})

			Context("when instance deprovisioning fails", func() {
				var instanceID string
				var provisionDetails map[string]any

				BeforeEach(func() {
					instanceID = uniqueInstanceID()
					provisionDetails = map[string]any{
						"plan_id":           "plan-id",
						"organization_guid": "organization-guid",
						"space_guid":        "space-guid",
					}
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				})

				Context("when an unexpected error occurs", func() {
					BeforeEach(func() {
						fakeServiceBroker.DeprovisionError = errors.New("broker failed")
					})

					It("returns a 500 with error", func() {
						response := makeInstanceDeprovisioningRequest(instanceID, "")

						Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{"description":"broker failed"}`))
					})

					It("logs an appropriate error", func() {
						makeInstanceDeprovisioningRequest(instanceID, "")
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "deprovision.unknown-error"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "broker failed"))
					})
				})

				Context("when a custom error occurs", func() {
					BeforeEach(func() {
						fakeServiceBroker.DeprovisionError = brokerapi.NewFailureResponse(
							errors.New("I failed in unique and interesting ways"),
							http.StatusTeapot,
							"interesting-failure",
						)
					})

					It("returns status teapot with error", func() {
						response := makeInstanceDeprovisioningRequest(instanceID, "")

						Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{"description":"I failed in unique and interesting ways"}`))
					})

					It("logs an appropriate error", func() {
						makeInstanceDeprovisioningRequest(instanceID, "")
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "deprovision.interesting-failure"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "I failed in unique and interesting ways"))
					})
				})
			})

			Context("the request is malformed", func() {
				It("missing header X-Broker-API-Version", func() {
					apiVersion = ""
					response := makeInstanceDeprovisioningRequestFull("instance-id", "service-id", "plan-id", "")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					apiVersion = "1.1"
					response := makeInstanceDeprovisioningRequestFull("instance-id", "service-id", "plan-id", "")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})

				It("missing service-id", func() {
					response := makeInstanceDeprovisioningRequestFull("instance-id", "", "plan-id", "")
					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "deprovision.service-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "service_id missing"))
				})

				It("missing plan-id", func() {
					response := makeInstanceDeprovisioningRequestFull("instance-id", "service-id", "", "")
					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "deprovision.plan-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "plan_id missing"))
				})
			})
		})

		Describe("getting instance", func() {
			It("returns the appropriate status code when it fails with a known error", func() {
				fakeServiceBroker.GetInstanceError = brokerapi.NewFailureResponse(errors.New("some error"), http.StatusUnprocessableEntity, "fire")

				response := makeGetInstanceRequest("instance-id")

				Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "getInstance.fire"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "some error"))
			})

			It("returns 500 when it fails with an unknown error", func() {
				fakeServiceBroker.GetInstanceError = errors.New("failed to get instance")

				response := makeGetInstanceRequest("instance-id")

				Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "getInstance.unknown-error"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "failed to get instance"))
			})

			Context("the request is malformed", func() {
				It("missing header X-Broker-API-Version", func() {
					apiVersion = ""
					response := makeGetInstanceRequest("instance-id")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					apiVersion = "1.1"
					response := makeGetInstanceRequest("instance-id")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})

				It("is using api version < 2.14", func() {
					apiVersion = "2.13"
					response := makeGetInstanceRequest("instance-id")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "getInstance.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "get instance endpoint only supported starting with OSB version 2.14"))
				})

				It("missing instance-id", func() {
					response := makeGetInstanceRequest("")
					Expect(response).To(HaveHTTPStatus(http.StatusNotFound))
				})
			})

			Context("fetch details", func() {
				It("returns 200 when service_id and plan_id are not provided", func() {
					response := makeGetInstanceWithQueryParamsRequest("instance-id", map[string]string{})

					Expect(response).To(HaveHTTPStatus(http.StatusOK))
					Expect(fakeServiceBroker.InstanceFetchDetails.ServiceID).To(Equal(""))
					Expect(fakeServiceBroker.InstanceFetchDetails.PlanID).To(Equal(""))
				})

				It("returns 200 when service_id and plan_id are provided", func() {
					params := map[string]string{
						"service_id": "e1307a5f-c54d-4f5d-924e-e5618c52ac0a",
						"plan_id":    "c6b2db23-60bf-4613-a91c-687372da42a5",
					}

					response := makeGetInstanceWithQueryParamsRequest("instance-id", params)

					Expect(response).To(HaveHTTPStatus(http.StatusOK))
					Expect(fakeServiceBroker.InstanceFetchDetails.ServiceID).To(Equal(params["service_id"]))
					Expect(fakeServiceBroker.InstanceFetchDetails.PlanID).To(Equal(params["plan_id"]))
				})

				It("returns 200 when only service_id is provided", func() {
					params := map[string]string{
						"service_id": "e1307a5f-c54d-4f5d-924e-e5618c52ac0a",
					}

					response := makeGetInstanceWithQueryParamsRequest("instance-id", params)

					Expect(response).To(HaveHTTPStatus(http.StatusOK))
					Expect(fakeServiceBroker.InstanceFetchDetails.ServiceID).To(Equal(params["service_id"]))
					Expect(fakeServiceBroker.InstanceFetchDetails.PlanID).To(Equal(""))
				})

				It("returns 200 when only plan_id is provided", func() {
					params := map[string]string{
						"plan_id": "c6b2db23-60bf-4613-a91c-687372da42a5",
					}

					response := makeGetInstanceWithQueryParamsRequest("instance-id", params)

					Expect(response).To(HaveHTTPStatus(http.StatusOK))
					Expect(fakeServiceBroker.InstanceFetchDetails.ServiceID).To(Equal(""))
					Expect(fakeServiceBroker.InstanceFetchDetails.PlanID).To(Equal(params["plan_id"]))
				})
			})
		})
	})

	Describe("binding lifecycle endpoint", func() {
		const bindingRequestIdentity = "Bind Request Identity Name"

		makeLastBindingOperationRequest := func(instanceID, bindingID string) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s/last_operation", instanceID, bindingID)

				buffer := &bytes.Buffer{}

				request, err := http.NewRequest("GET", path, buffer)

				Expect(err).NotTo(HaveOccurred())

				request.Header.Add("X-Broker-Api-Version", "2.14")
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")
				request.Header.Add("X-Broker-API-Request-Identity", bindingRequestIdentity)

				response = r.Do(request)
			})
			return response
		}

		makeGetBindingWithQueryParamsRequest := func(instanceID, bindingID string, params map[string]string) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)

				buffer := &bytes.Buffer{}

				request, err := http.NewRequest("GET", path, buffer)

				Expect(err).NotTo(HaveOccurred())

				request.Header.Add("X-Broker-Api-Version", "2.14")
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")

				q := request.URL.Query()
				for query, value := range params {
					q.Add(query, value)
				}
				request.URL.RawQuery = q.Encode()
				response = r.Do(request)
			})
			return response
		}

		makeGetBindingRequestWithSpecificAPIVersion := func(instanceID, bindingID string, apiVersion string) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)

				buffer := &bytes.Buffer{}

				request, err := http.NewRequest("GET", path, buffer)

				Expect(err).NotTo(HaveOccurred())

				if apiVersion != "" {
					request.Header.Add("X-Broker-Api-Version", apiVersion)
				}
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")
				request.Header.Add("X-Broker-API-Request-Identity", bindingRequestIdentity)

				response = r.Do(request)
			})
			return response
		}

		makeBindingRequestWithSpecificAPIVersion := func(instanceID, bindingID string, details map[string]any, apiVersion string, async bool) (response *http.Response) {
			withServer(brokerAPI, func(r requester) {
				path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s?accepts_incomplete=%v",
					instanceID, bindingID, async)

				buffer := &bytes.Buffer{}

				if details != nil {
					json.NewEncoder(buffer).Encode(details)
				}

				request, err := http.NewRequest("PUT", path, buffer)

				Expect(err).NotTo(HaveOccurred())

				if apiVersion != "" {
					request.Header.Add("X-Broker-Api-Version", apiVersion)
				}
				request.Header.Add("Content-Type", "application/json")
				request.SetBasicAuth("username", "password")
				request.Header.Add("X-Broker-API-Request-Identity", bindingRequestIdentity)

				response = r.Do(request)
			})
			return response
		}

		makeBindingRequest := func(instanceID, bindingID string, details map[string]any) *http.Response {
			return makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, details, "2.10", false)
		}

		makeAsyncBindingRequest := func(instanceID, bindingID string, details map[string]any) *http.Response {
			return makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, details, "2.14", true)
		}

		Describe("binding", func() {
			var (
				instanceID string
				bindingID  string
				details    map[string]any
			)

			BeforeEach(func() {
				instanceID = uniqueInstanceID()
				bindingID = uniqueBindingID()
				details = map[string]any{
					"app_guid":   "app_guid",
					"plan_id":    "plan_id",
					"service_id": "service_id",
					"parameters": map[string]any{
						"new-param": "new-param-value",
					},
				}
			})

			When("can bind", func() {
				It("calls Bind on the service broker with the instance and binding ids", func() {
					makeBindingRequest(instanceID, bindingID, details)
					Expect(fakeServiceBroker.BoundInstanceIDs).To(ContainElement(instanceID))
					_, ok := fakeServiceBroker.BoundBindings[bindingID]
					Expect(ok).To(BeTrue())
					Expect(fakeServiceBroker.BoundBindings[bindingID]).To(Equal(brokerapi.BindDetails{
						AppGUID:       "app_guid",
						PlanID:        "plan_id",
						ServiceID:     "service_id",
						RawParameters: json.RawMessage(`{"new-param":"new-param-value"}`),
					}))
				})

				It("calls bind with details with raw parameters", func() {
					makeBindingRequest(instanceID, bindingID, details)
					detailsWithRawParameters := brokerapi.DetailsWithRawParameters(fakeServiceBroker.BoundBindings[bindingID])
					rawParameters := detailsWithRawParameters.GetRawParameters()
					Expect(rawParameters).To(Equal(json.RawMessage(`{"new-param":"new-param-value"}`)))
				})

				It("returns a 201 with body", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)

					Expect(response).To(HaveHTTPStatus(http.StatusCreated))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(fixture("binding.json")))
				})

				Context("when syslog_drain_url is being passed", func() {
					BeforeEach(func() {
						fakeServiceBroker.SyslogDrainURL = "some-drain-url"
					})

					It("responds with the syslog drain url", func() {
						response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
						Expect(readBody(response)).To(MatchJSON(fixture("binding_with_syslog.json")))
					})
				})

				Context("when route_service_url is being passed", func() {
					BeforeEach(func() {
						fakeServiceBroker.RouteServiceURL = "some-route-url"
					})

					It("responds with the route service url", func() {
						response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
						Expect(readBody(response)).To(MatchJSON(fixture("binding_with_route_service.json")))
					})
				})

				Context("when a volume mount is being passed", func() {
					BeforeEach(func() {
						fakeServiceBroker.VolumeMounts = []brokerapi.VolumeMount{{
							Driver:       "driver",
							ContainerDir: "/dev/null",
							Mode:         "rw",
							DeviceType:   "shared",
							Device: brokerapi.SharedDevice{
								VolumeId:    "some-guid",
								MountConfig: map[string]any{"key": "value"},
							},
						}}
					})

					Context("when the broker API version is greater than 2.9", func() {
						It("responds with a volume mount", func() {
							response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
							Expect(readBody(response)).To(MatchJSON(fixture("binding_with_volume_mounts.json")))
						})
					})

					Context("when the broker API version is 2.9", func() {
						It("responds with an experimental volume mount", func() {
							response := makeBindingRequestWithSpecificAPIVersion(uniqueInstanceID(), uniqueBindingID(), details, "2.9", false)
							Expect(readBody(response)).To(MatchJSON(fixture("binding_with_experimental_volume_mounts.json")))
						})
					})

					Context("when the broker API version is 2.8", func() {
						It("responds with an experimental volume mount", func() {
							response := makeBindingRequestWithSpecificAPIVersion(uniqueInstanceID(), uniqueBindingID(), details, "2.8", false)
							Expect(readBody(response)).To(MatchJSON(fixture("binding_with_experimental_volume_mounts.json")))
						})
					})
				})

				Context("when no bind details are being passed", func() {
					It("returns a 422", func() {
						response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), nil)
						Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					})
				})

				Context("when there are arbitrary params", func() {
					var (
						rawParams string
						rawCtx    string
					)

					BeforeEach(func() {
						details["parameters"] = map[string]any{
							"string": "some-string",
							"number": 1,
							"object": struct{ Name string }{"some-name"},
							"array":  []any{"a", "b", "c"},
						}

						details["context"] = map[string]any{
							"platform":      "fake-platform",
							"serial-number": 12648430,
							"object":        struct{ Name string }{"parameter"},
							"array":         []any{"1", "2", "3"},
						}

						rawParams = `{
							"string":"some-string",
							"number":1,
							"object": { "Name": "some-name" },
							"array": [ "a", "b", "c" ]
						}`
						rawCtx = `{
							"platform":"fake-platform",
							"serial-number":12648430,
							"object": {"Name":"parameter"},
							"array":[ "1", "2", "3" ]
						}`
					})

					It("calls Bind on the service broker with all params", func() {
						makeBindingRequest(instanceID, bindingID, details)
						Expect(string(fakeServiceBroker.BoundBindings[bindingID].RawParameters)).To(MatchJSON(rawParams))
					})

					It("calls Bind with details with raw parameters", func() {
						makeBindingRequest(instanceID, bindingID, details)
						detailsWithRawParameters := brokerapi.DetailsWithRawParameters(fakeServiceBroker.BoundBindings[bindingID])
						rawParameters := detailsWithRawParameters.GetRawParameters()
						Expect(string(rawParameters)).To(MatchJSON(rawParams))
					})

					It("calls Bind with details with raw context", func() {
						makeBindingRequest(instanceID, bindingID, details)
						detailsWithRawContext := brokerapi.DetailsWithRawContext(fakeServiceBroker.BoundBindings[bindingID])
						rawContext := detailsWithRawContext.GetRawContext()
						Expect(string(rawContext)).To(MatchJSON(rawCtx))
					})
				})

				When("there are details in the bind_resource", func() {

					It("calls Bind on the service broker with the bind_resource", func() {

						details["bind_resource"] = map[string]any{
							"app_guid":             "a-guid",
							"space_guid":           "a-space-guid",
							"route":                "route.cf-apps.com",
							"credential_client_id": "some-credentials",
						}

						makeBindingRequest(instanceID, bindingID, details)
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource).NotTo(BeNil())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.AppGuid).To(Equal("a-guid"))
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.SpaceGuid).To(Equal("a-space-guid"))
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.Route).To(Equal("route.cf-apps.com"))
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.CredentialClientID).To(Equal("some-credentials"))
					})
				})

				When("there are no details in the bind_resource", func() {

					It("calls Bind on the service broker with an empty bind_resource", func() {

						details["bind_resource"] = map[string]any{}

						makeBindingRequest(instanceID, bindingID, details)
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource).NotTo(BeNil())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.AppGuid).To(BeEmpty())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.SpaceGuid).To(BeEmpty())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.Route).To(BeEmpty())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.CredentialClientID).To(BeEmpty())
					})
				})

				When("backup_agent is requested", func() {
					BeforeEach(func() {
						details["bind_resource"] = map[string]any{"backup_agent": true}
						fakeServiceBroker.BackupAgentURL = "http://backup.example.com"
					})

					It("responds with the backup agent url", func() {
						response := makeBindingRequest(instanceID, bindingID, details)
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource).NotTo(BeNil())
						Expect(fakeServiceBroker.BoundBindings[bindingID].BindResource.BackupAgent).To(BeTrue())

						Expect(response).To(HaveHTTPStatus(http.StatusCreated))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{"backup_agent_url":"http://backup.example.com"}`))
					})
				})
			})

			Context("when the associated instance does not exist", func() {
				var instanceID string

				BeforeEach(func() {
					fakeServiceBroker.BindError = brokerapi.ErrInstanceDoesNotExist
				})

				It("returns a 404 with error", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(response).To(HaveHTTPStatus(http.StatusNotFound))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"instance does not exist"}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeBindingRequest(instanceID, uniqueBindingID(), details)
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.instance-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance does not exist"))
				})
			})

			Context("when the requested binding already exists", func() {
				var instanceID, bindingID string

				BeforeEach(func() {
					fakeServiceBroker.BindError = brokerapi.ErrBindingAlreadyExists
				})

				Context("returns a statusOK", func() {
					BeforeEach(func() {
						fakeServiceBroker.BindError = nil

						instanceID = uniqueInstanceID()
						bindingID = uniqueBindingID()

						makeBindingRequest(instanceID, bindingID, details)
					})

					It("sync broker response", func() {
						response := makeBindingRequest(instanceID, bindingID, details)
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					})

					It("async broker response", func() {
						fakeAsyncServiceBroker := &fakes.FakeAsyncServiceBroker{
							FakeServiceBroker: *fakeServiceBroker,
						}
						brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))

						response := makeAsyncBindingRequest(instanceID, bindingID, details)
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					})
				})

				It("returns a statusConflict", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(response).To(HaveHTTPStatus(http.StatusConflict))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
				})

				It("returns an error JSON object", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(readBody(response)).To(MatchJSON(`{"description":"binding already exists"}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeBindingRequest(instanceID, uniqueBindingID(), details)
					makeBindingRequest(instanceID, uniqueBindingID(), details)

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.binding-already-exists"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "binding already exists"))
				})
			})

			Context("when the binding returns an unknown error", func() {
				BeforeEach(func() {
					fakeServiceBroker.BindError = errors.New("unknown error")
				})

				It("returns a generic 500 error response", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"unknown error"}`))
				})

				It("logs a detailed error message", func() {
					makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.unknown-error"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "unknown error"))
				})
			})

			Context("when the binding returns a custom error", func() {
				BeforeEach(func() {
					fakeServiceBroker.BindError = brokerapi.NewFailureResponse(
						errors.New("I failed in unique and interesting ways"),
						http.StatusTeapot,
						"interesting-failure",
					)
				})

				It("returns status teapot and error", func() {
					response := makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"I failed in unique and interesting ways"}`))
				})

				It("logs an appropriate error", func() {
					makeBindingRequest(uniqueInstanceID(), uniqueBindingID(), details)
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.interesting-failure"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "I failed in unique and interesting ways"))
				})
			})

			Context("when an async binding is requested", func() {
				var (
					fakeAsyncServiceBroker *fakes.FakeAsyncServiceBroker
				)

				BeforeEach(func() {
					fakeAsyncServiceBroker = &fakes.FakeAsyncServiceBroker{
						FakeServiceBroker: *fakeServiceBroker,
					}
					brokerAPI = brokerapi.NewWithOptions(fakeAsyncServiceBroker, brokerLogger, brokerapi.WithBrokerCredentials(credentials))
				})

				When("the api version is < 2.14", func() {
					It("successfully returns a sync binding response", func() {
						response := makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, details, "2.13", true)
						Expect(response).To(HaveHTTPStatus(http.StatusCreated))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(fixture("binding.json")))
					})

					It("fails for GetBinding request", func() {
						response := makeGetBindingRequestWithSpecificAPIVersion(instanceID, bindingID, "1.13")
						Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					})
				})

				When("the api version is 2.14", func() {
					It("returns an appropriate status code and operation data", func() {
						response := makeAsyncBindingRequest(instanceID, bindingID, details)
						Expect(response).To(HaveHTTPStatus(http.StatusAccepted))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(fixture("async_bind_response.json")))
					})

					It("can be polled with lastBindingOperation", func() {
						fakeAsyncServiceBroker.LastOperationState = "succeeded"
						fakeAsyncServiceBroker.LastOperationDescription = "some description"
						response := makeLastBindingOperationRequest(instanceID, bindingID)
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(fixture("last_operation_succeeded.json")))
					})

					It("returns the binding for the async request on getBinding", func() {
						response := makeGetBindingRequestWithSpecificAPIVersion(instanceID, bindingID, "2.14")
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(fixture("binding.json")))
					})
				})
			})

			Context("the request is malformed", func() {
				BeforeEach(func() {
					bindingID = uniqueBindingID()
				})

				It("missing header X-Broker-API-Version", func() {
					response := makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, map[string]any{}, "", false)
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					response := makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, map[string]any{}, "1.14", false)
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})

				It("missing service-id", func() {
					response := makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, map[string]any{"plan_id": "123"}, "2.14", false)
					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.service-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "service_id missing"))
				})

				It("missing plan-id", func() {
					response := makeBindingRequestWithSpecificAPIVersion(instanceID, bindingID, map[string]any{"service_id": "123"}, "2.14", false)
					Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "bind.plan-id-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "plan_id missing"))
				})
			})
		})

		Describe("unbinding", func() {
			const unbindRequestIdentity = "Unbind Request Identity Name"

			makeUnbindingRequestWithServiceIDPlanID := func(instanceID, bindingID, serviceID, planID, apiVersion string) (response *http.Response) {
				withServer(brokerAPI, func(r requester) {
					path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s?plan_id=%s&service_id=%s",
						instanceID, bindingID, planID, serviceID)
					request := must(http.NewRequest("DELETE", path, strings.NewReader("")))
					request.Header.Add("Content-Type", "application/json")
					request.Header.Add("X-Broker-API-Version", apiVersion)
					request.SetBasicAuth("username", "password")
					request.Header.Add("X-Broker-API-Request-Identity", unbindRequestIdentity)

					response = r.Do(request)
				})
				return response
			}

			makeUnbindingRequest := func(instanceID string, bindingID string) *http.Response {
				return makeUnbindingRequestWithServiceIDPlanID(instanceID, bindingID, "service-id", "plan-id", "2.13")
			}

			Context("when the associated instance exists", func() {
				var instanceID string
				var provisionDetails map[string]any

				BeforeEach(func() {
					instanceID = uniqueInstanceID()
					provisionDetails = map[string]any{
						"service_id":        fakeServiceBroker.ServiceID,
						"plan_id":           "plan-id",
						"organization_guid": "organization-guid",
						"space_guid":        "space-guid",
					}
					makeInstanceProvisioningRequest(instanceID, provisionDetails, "")
				})

				Context("the request is malformed", func() {
					var bindingID string

					BeforeEach(func() {
						bindingID = uniqueBindingID()
						makeBindingRequest(instanceID, bindingID, map[string]any{})
					})

					It("missing header X-Broker-API-Version", func() {
						response := makeUnbindingRequestWithServiceIDPlanID(instanceID, bindingID, "service-id", "plan-id", "")
						Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
					})

					It("has wrong version of API", func() {
						response := makeUnbindingRequestWithServiceIDPlanID(instanceID, bindingID, "service-id", "plan-id", "1.1")
						Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
					})

					It("missing service-id", func() {
						response := makeUnbindingRequestWithServiceIDPlanID(instanceID, bindingID, "", "plan-id", "2.13")
						Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.service-id-missing"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "service_id missing"))
					})

					It("missing plan-id", func() {
						response := makeUnbindingRequestWithServiceIDPlanID(instanceID, bindingID, "service-id", "", "2.13")
						Expect(response).To(HaveHTTPStatus(http.StatusBadRequest))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.plan-id-missing"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "plan_id missing"))
					})
				})

				Context("and the binding exists", func() {
					var bindingID string

					BeforeEach(func() {
						bindingID = uniqueBindingID()
						makeBindingRequest(instanceID, bindingID, map[string]any{
							"service_id": "service_id", "plan_id": "plan_id",
						})
					})

					It("returns a 200", func() {
						response := makeUnbindingRequest(instanceID, bindingID)
						Expect(response).To(HaveHTTPStatus(http.StatusOK))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{}`))
					})

					It("contains plan_id", func() {
						makeUnbindingRequest(instanceID, bindingID)
						Expect(fakeServiceBroker.UnbindingDetails.PlanID).To(Equal("plan-id"))
					})

					It("contains service_id", func() {
						makeUnbindingRequest(instanceID, bindingID)
						Expect(fakeServiceBroker.UnbindingDetails.ServiceID).To(Equal("service-id"))
					})
				})

				Context("but the binding does not exist", func() {
					It("returns a 410", func() {
						response := makeUnbindingRequest(instanceID, "does-not-exist")
						Expect(response).To(HaveHTTPStatus(http.StatusGone))
						Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
						Expect(readBody(response)).To(MatchJSON(`{}`))
					})

					It("logs an appropriate error message", func() {
						makeUnbindingRequest(instanceID, "does-not-exist")

						Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.binding-missing"))
						Expect(lastLogLine()).To(HaveKeyWithValue("error", "binding does not exist"))
					})
				})
			})

			Context("when the associated instance does not exist", func() {
				var instanceID string

				It("returns a 410", func() {
					response := makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response).To(HaveHTTPStatus(http.StatusGone))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{}`))
				})

				It("logs an appropriate error", func() {
					instanceID = uniqueInstanceID()
					makeUnbindingRequest(instanceID, uniqueBindingID())

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.instance-missing"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance does not exist"))
				})
			})

			Context("when unbinding returns an unknown error", func() {
				BeforeEach(func() {
					fakeServiceBroker.UnbindError = errors.New("unknown error")
				})

				It("returns a generic 500 error response", func() {
					response := makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"unknown error"}`))
				})

				It("logs a detailed error message", func() {
					makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.unknown-error"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "unknown error"))
				})
			})

			Context("when unbinding returns a custom error", func() {
				BeforeEach(func() {
					fakeServiceBroker.UnbindError = brokerapi.NewFailureResponse(
						errors.New("I failed in unique and interesting ways"),
						http.StatusTeapot,
						"interesting-failure",
					)
				})

				It("returns status teapot with error", func() {
					response := makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(unbindRequestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"I failed in unique and interesting ways"}`))
				})

				It("logs an appropriate error", func() {
					makeUnbindingRequest(uniqueInstanceID(), uniqueBindingID())
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "unbind.interesting-failure"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "I failed in unique and interesting ways"))
				})
			})
		})

		Describe("last_operation", func() {
			makeLastOperationRequest := func(instanceID, operationData, apiVersion string) (response *http.Response) {
				withServer(brokerAPI, func(r requester) {
					path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
					if operationData != "" {
						path = fmt.Sprintf("%s?operation=%s", path, url.QueryEscape(operationData))
					}

					request := must(http.NewRequest("GET", path, strings.NewReader("")))
					if apiVersion != "" {
						request.Header.Add("X-Broker-API-Version", apiVersion)
					}
					request.Header.Add("Content-Type", "application/json")
					request.SetBasicAuth("username", "password")
					request.Header.Add("X-Broker-API-Request-Identity", requestIdentity)
					response = r.Do(request)
				})
				return response
			}

			It("calls the broker with the relevant instance ID", func() {
				instanceID := "instanceID"
				makeLastOperationRequest(instanceID, "", "2.14")
				Expect(fakeServiceBroker.LastOperationInstanceID).To(Equal(instanceID))
			})

			It("calls the broker with the URL decoded operation data", func() {
				instanceID := "an-instance"
				operationData := `{"foo":"bar"}`
				makeLastOperationRequest(instanceID, operationData, "2.14")
				Expect(fakeServiceBroker.LastOperationData).To(Equal(operationData))
			})

			It("should return succeeded if the operation completed successfully", func() {
				fakeServiceBroker.LastOperationState = "succeeded"
				fakeServiceBroker.LastOperationDescription = "some description"

				instanceID := "instanceID"
				response := makeLastOperationRequest(instanceID, "", "2.14")

				var logs []map[string]any
				for i, line := range strings.Split(strings.TrimSpace(string(logBuffer.Contents())), "\n") {
					var receiver map[string]any
					Expect(json.Unmarshal([]byte(line), &receiver)).To(Succeed(), fmt.Sprintf("line %d", i))
					logs = append(logs, receiver)
				}

				Expect(logs[0]).To(HaveKeyWithValue("msg", "lastOperation.starting-check-for-operation"))
				Expect(logs[0]).To(HaveKeyWithValue("instance-id", instanceID))

				Expect(logs[1]).To(HaveKeyWithValue("msg", "lastOperation.done-check-for-operation"))
				Expect(logs[1]).To(HaveKeyWithValue("instance-id", instanceID))
				Expect(logs[1]).To(HaveKeyWithValue("state", BeEquivalentTo(fakeServiceBroker.LastOperationState)))

				Expect(response).To(HaveHTTPStatus(http.StatusOK))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(readBody(response)).To(MatchJSON(fixture("last_operation_succeeded.json")))
			})

			It("should return a 410 and log in case the instance id is not found", func() {
				fakeServiceBroker.LastOperationError = brokerapi.ErrInstanceDoesNotExist
				instanceID := "non-existing"
				response := makeLastOperationRequest(instanceID, "", "2.14")

				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "lastOperation.instance-missing"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "instance does not exist"))

				Expect(response).To(HaveHTTPStatus(http.StatusGone))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
				Expect(readBody(response)).To(MatchJSON(`{}`))
			})

			Context("when last_operation returns an unknown error", func() {
				BeforeEach(func() {
					fakeServiceBroker.LastOperationError = errors.New("unknown error")
				})

				It("returns a generic 500 error response", func() {
					response := makeLastOperationRequest("instanceID", "", "2.14")

					Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description": "unknown error"}`))
				})

				It("logs a detailed error message", func() {
					makeLastOperationRequest("instanceID", "", "2.14")

					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "lastOperation.unknown-error"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "unknown error"))
				})
			})

			Context("when last_operation returns a custom error", func() {
				BeforeEach(func() {
					fakeServiceBroker.LastOperationError = brokerapi.NewFailureResponse(
						errors.New("I failed in unique and interesting ways"),
						http.StatusTeapot,
						"interesting-failure",
					)
				})

				It("returns status teapot with error", func() {
					response := makeLastOperationRequest("instanceID", "", "2.14")
					Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(readBody(response)).To(MatchJSON(`{"description":"I failed in unique and interesting ways"}`))
				})

				It("logs an appropriate error", func() {
					makeLastOperationRequest("instanceID", "", "2.14")
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "lastOperation.interesting-failure"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "I failed in unique and interesting ways"))
				})
			})

			Context("the request is malformed", func() {
				It("missing header X-Broker-API-Version", func() {
					response := makeLastOperationRequest("instance-id", "", "")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header not set"))
				})

				It("has wrong version of API", func() {
					response := makeLastOperationRequest("instance-id", "", "1.2")
					Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
					Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(requestIdentity))
					Expect(lastLogLine()).To(HaveKeyWithValue("msg", "version-header-check.broker-api-version-invalid"))
					Expect(lastLogLine()).To(HaveKeyWithValue("error", "X-Broker-API-Version Header must be 2.x"))
				})
			})
		})

		Describe("get binding", func() {
			It("responds with 500 when the broker fails with an unknown error", func() {
				fakeServiceBroker.GetBindingError = errors.New("something failed")

				response := makeGetBindingRequestWithSpecificAPIVersion("some-instance", "some-binding", "2.14")
				Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "getBinding.unknown-error"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "something failed"))
			})

			It("returns the appropriate status code when it fails with a known error", func() {
				fakeServiceBroker.GetBindingError = brokerapi.NewFailureResponse(errors.New("some error"), http.StatusUnprocessableEntity, "fire")

				response := makeGetBindingRequestWithSpecificAPIVersion("some-instance", "some-binding", "2.14")
				Expect(response).To(HaveHTTPStatus(http.StatusUnprocessableEntity))
				Expect(response.Header.Get("X-Broker-API-Request-Identity")).To(Equal(bindingRequestIdentity))
				Expect(lastLogLine()).To(HaveKeyWithValue("msg", "getBinding.fire"))
				Expect(lastLogLine()).To(HaveKeyWithValue("error", "some error"))
			})
		})

		Context("fetch details", func() {
			It("returns 200 when service_id and plan_id are not provided", func() {
				response := makeGetBindingWithQueryParamsRequest("instance-id", "binding-id", map[string]string{})

				Expect(response).To(HaveHTTPStatus(http.StatusOK))
				Expect(fakeServiceBroker.BindingFetchDetails.ServiceID).To(Equal(""))
				Expect(fakeServiceBroker.BindingFetchDetails.PlanID).To(Equal(""))
			})

			It("returns 200 when service_id and plan_id are provided", func() {
				params := map[string]string{
					"service_id": "e1307a5f-c54d-4f5d-924e-e5618c52ac0a",
					"plan_id":    "c6b2db23-60bf-4613-a91c-687372da42a5",
				}

				response := makeGetBindingWithQueryParamsRequest("instance-id", "binding-id", params)

				Expect(response).To(HaveHTTPStatus(http.StatusOK))
				Expect(fakeServiceBroker.BindingFetchDetails.ServiceID).To(Equal(params["service_id"]))
				Expect(fakeServiceBroker.BindingFetchDetails.PlanID).To(Equal(params["plan_id"]))
			})

			It("returns 200 when only service_id is provided", func() {
				params := map[string]string{
					"service_id": "e1307a5f-c54d-4f5d-924e-e5618c52ac0a",
				}

				response := makeGetBindingWithQueryParamsRequest("instance-id", "binding-id", params)

				Expect(response).To(HaveHTTPStatus(http.StatusOK))
				Expect(fakeServiceBroker.BindingFetchDetails.ServiceID).To(Equal(params["service_id"]))
				Expect(fakeServiceBroker.BindingFetchDetails.PlanID).To(Equal(""))
			})

			It("returns 200 when only plan_id is provided", func() {
				params := map[string]string{
					"plan_id": "c6b2db23-60bf-4613-a91c-687372da42a5",
				}

				response := makeGetBindingWithQueryParamsRequest("instance-id", "binding-id", params)

				Expect(response).To(HaveHTTPStatus(http.StatusOK))
				Expect(fakeServiceBroker.BindingFetchDetails.ServiceID).To(Equal(""))
				Expect(fakeServiceBroker.BindingFetchDetails.PlanID).To(Equal(params["plan_id"]))
			})
		})
	})

	Describe("NewWithOptions()", func() {
		var provisionDetails map[string]any

		BeforeEach(func() {
			provisionDetails = map[string]any{
				"service_id":        fakeServiceBroker.ServiceID,
				"plan_id":           "plan-id",
				"organization_guid": "organization-guid",
				"space_guid":        "space-guid",
			}
		})

		Describe("WithAdditionalMiddleware()", func() {
			It("adds additional middleware", func() {
				const (
					customMiddlewareError = "fake custom middleware error"
					customMiddlewareCode  = http.StatusTeapot
				)

				By("adding some custom middleware that fails")
				brokerAPI = brokerapi.NewWithOptions(fakeServiceBroker, brokerLogger, brokerapi.WithAdditionalMiddleware(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						http.Error(w, customMiddlewareError, customMiddlewareCode)
					})
				}))

				By("checking for the specific failure from the custom middleware")
				response := makeInstanceProvisioningRequest(uniqueInstanceID(), provisionDetails, "")
				Expect(response).To(HaveHTTPStatus(customMiddlewareCode))
				Expect(readBody(response)).To(Equal(customMiddlewareError + "\n"))
			})
		})

		It("will accept URL-encoded paths", func() {
			const encodedInstanceID = "foo%2Fbar"
			brokerAPI = brokerapi.New(fakeServiceBroker, brokerLogger, credentials)
			response := makeInstanceProvisioningRequest(encodedInstanceID, provisionDetails, "")
			Expect(response).To(HaveHTTPStatus(http.StatusCreated))
			Expect(fakeServiceBroker.ProvisionedInstances).To(HaveKey("foo/bar"))
		})
	})
})

func must[A any](input A, err error) A {
	GinkgoHelper()

	Expect(err).NotTo(HaveOccurred())
	return input
}

func readBody(res *http.Response) string {
	GinkgoHelper()

	body := must(io.ReadAll(res.Body))
	res.Body.Close()
	return string(body)
}

func unmarshalBody(res *http.Response) (body map[string]string) {
	GinkgoHelper()

	Expect(json.Unmarshal([]byte(readBody(res)), &body)).To(Succeed())
	return
}

type requester func(*http.Request) *http.Response

func (r requester) Do(req *http.Request) *http.Response {
	return r(req)
}

func withServer(handler http.Handler, callback func(requester)) {
	server := httptest.NewServer(handler)
	callback(func(request *http.Request) *http.Response {
		request.URL = must(url.Parse(fmt.Sprintf("%s%s", server.URL, request.URL)))
		return must(server.Client().Do(request))
	})
}
