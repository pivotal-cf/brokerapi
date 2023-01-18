package brokerapi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v8"
)

var _ = Describe("MaintenanceInfo", func() {
	Describe("Equals", func() {
		DescribeTable(
			"returns false",
			func(m1, m2 brokerapi.MaintenanceInfo) {
				Expect(m1.Equals(m2)).To(BeFalse())
			},
			Entry(
				"one property is missing",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
				}),
			Entry(
				"one extra property is added",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
				},
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				}),
			Entry(
				"one property is different",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test-not-the-same",
					Version: "1.2.3",
				}),
			Entry(
				"all properties are missing in one of the objects",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				brokerapi.MaintenanceInfo{}),
			Entry(
				"all properties are defined but different",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"bar": "foo"},
					Private: "test-not-the-same",
					Version: "8.9.6-rc3",
				}),
		)

		DescribeTable(
			"returns true",
			func(m1, m2 brokerapi.MaintenanceInfo) {
				Expect(m1.Equals(m2)).To(BeTrue())
			},
			Entry(
				"all properties are the same",
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				brokerapi.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				}),
			Entry(
				"all properties are empty",
				brokerapi.MaintenanceInfo{},
				brokerapi.MaintenanceInfo{}),
			Entry(
				"both struct's are nil",
				nil,
				nil),
		)
	})
})
