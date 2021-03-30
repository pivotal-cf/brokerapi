package domain_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

var maximumPollingDuration = 3600

var _ = Describe("ServiceCatalog", func() {
	Describe("Service", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				service := domain.Service{
					ID:            "ID-1",
					Name:          "Cassandra",
					Description:   "A Cassandra Plan",
					Bindable:      true,
					Plans:         []domain.ServicePlan{},
					Metadata:      &domain.ServiceMetadata{},
					Tags:          []string{"test"},
					PlanUpdatable: true,
					DashboardClient: &domain.ServiceDashboardClient{
						ID:          "Dashboard ID",
						Secret:      "dashboardsecret",
						RedirectURI: "the.dashboa.rd",
					},
					AllowContextUpdates: true,
				}
				jsonString := `{
					"id":"ID-1",
				  	"name":"Cassandra",
					"description":"A Cassandra Plan",
					"bindable":true,
					"plan_updateable":true,
					"tags":["test"],
					"plans":[],
                    "allow_context_updates":true,
					"dashboard_client":{
						"id":"Dashboard ID",
						"secret":"dashboardsecret",
						"redirect_uri":"the.dashboa.rd"
					},
					"metadata":{

					}
				}`
				Expect(json.Marshal(service)).To(MatchJSON(jsonString))
			})
		})

		It("encodes the optional 'requires' fields", func() {
			service := domain.Service{
				ID:            "ID-1",
				Name:          "Cassandra",
				Description:   "A Cassandra Plan",
				Bindable:      true,
				Plans:         []domain.ServicePlan{},
				Metadata:      &domain.ServiceMetadata{},
				Tags:          []string{"test"},
				PlanUpdatable: true,
				Requires: []domain.RequiredPermission{
					domain.PermissionRouteForwarding,
					domain.PermissionSyslogDrain,
					domain.PermissionVolumeMount,
				},
				DashboardClient: &domain.ServiceDashboardClient{
					ID:          "Dashboard ID",
					Secret:      "dashboardsecret",
					RedirectURI: "the.dashboa.rd",
				},
			}
			jsonString := `{
				"id":"ID-1",
					"name":"Cassandra",
				"description":"A Cassandra Plan",
				"bindable":true,
				"plan_updateable":true,
				"tags":["test"],
				"plans":[],
				"requires": ["route_forwarding", "syslog_drain", "volume_mount"],
				"dashboard_client":{
					"id":"Dashboard ID",
					"secret":"dashboardsecret",
					"redirect_uri":"the.dashboa.rd"
				},
				"metadata":{

				}
			}`
			Expect(json.Marshal(service)).To(MatchJSON(jsonString))
		})
	})

	Describe("ServicePlan", func() {
		Describe("JSON encoding", func() {
			It("uses the correct keys", func() {
				plan := domain.ServicePlan{
					ID:            "ID-1",
					Name:          "Cassandra",
					Description:   "A Cassandra Plan",
					Bindable:      domain.BindableValue(true),
					Free:          domain.FreeValue(true),
					PlanUpdatable: domain.PlanUpdatableValue(true),
					Metadata: &domain.ServicePlanMetadata{
						Bullets:     []string{"hello", "its me"},
						DisplayName: "name",
					},
					MaximumPollingDuration: &maximumPollingDuration,
					MaintenanceInfo: &domain.MaintenanceInfo{
						Public: map[string]string{
							"name": "foo",
						},
						Private:     "someprivatehashedvalue",
						Version:     "8.1.0",
						Description: "test",
					},
				}
				jsonString := `{
					"id":"ID-1",
					"name":"Cassandra",
					"description":"A Cassandra Plan",
					"free": true,
					"bindable": true,
					"metadata":{
						"bullets":["hello", "its me"],
						"displayName":"name"
					},
					"plan_updateable": true,
					"maximum_polling_duration": 3600,
					"maintenance_info": {
						"public": {
							"name": "foo"
						},
						"private": "someprivatehashedvalue",
						"version": "8.1.0",
						"description": "test"
					}
				}`

				Expect(json.Marshal(plan)).To(MatchJSON(jsonString))
			})
		})
	})
})
