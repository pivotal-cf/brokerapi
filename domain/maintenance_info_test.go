package domain_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/domain"
)

var _ = Describe("MaintenanceInfo", func() {
	Describe("Equals", func() {
		DescribeTable(
			"returns false",
			func(m1, m2 domain.MaintenanceInfo) {
				Expect(m1.Equals(m2)).To(BeFalse())
			},
			Entry(
				"one property is missing",
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
				}),
			Entry(
				"one extra property is added",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Description: "test",
				},
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				}),
			Entry(
				"one property is different",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test-different",
				},
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test-not-the-same",
					Version:     "1.2.3",
					Description: "test",
				}),
			Entry(
				"all properties are missing in one of the objects",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				},
				domain.MaintenanceInfo{}),
			Entry(
				"all properties are defined but different",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				},
				domain.MaintenanceInfo{
					Public:      map[string]string{"bar": "foo"},
					Private:     "test-not-the-same",
					Version:     "8.9.6-rc3",
					Description: "test-different",
				}),
		)

		DescribeTable(
			"returns true",
			func(m1, m2 domain.MaintenanceInfo) {
				Expect(m1.Equals(m2)).To(BeTrue())
			},
			Entry(
				"all properties are the same",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				},
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				}),
			Entry(
				"all properties are empty",
				domain.MaintenanceInfo{},
				domain.MaintenanceInfo{}),
			Entry(
				"both struct's are nil",
				nil,
				nil),
		)
	})
})
