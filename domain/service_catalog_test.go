package domain_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/domain"
)

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
				}
				jsonString := `{
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
					ID:          "ID-1",
					Name:        "Cassandra",
					Description: "A Cassandra Plan",
					Bindable:    domain.BindableValue(true),
					Free:        domain.FreeValue(true),
					Metadata: &domain.ServicePlanMetadata{
						Bullets:     []string{"hello", "its me"},
						DisplayName: "name",
					},
					MaintenanceInfo: &domain.MaintenanceInfo{
						Public: map[string]string{
							"name": "foo",
						},
						Private: "someprivatehashedvalue",
						Version: "8.1.0",
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
					"maintenance_info": {
						"public": {
							"name": "foo"
						},
						"private": "someprivatehashedvalue",
						"version": "8.1.0"
					}
				}`

				Expect(json.Marshal(plan)).To(MatchJSON(jsonString))
			})
		})
	})
})
