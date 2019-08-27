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
			Entry("public field is different",
				domain.MaintenanceInfo{Public: map[string]string{"foo": "bar"}},
				domain.MaintenanceInfo{Public: map[string]string{"foo": "foo"}},
			),
			Entry("private field is different",
				domain.MaintenanceInfo{Private: "foo"},
				domain.MaintenanceInfo{Private: "bar"},
			),
			Entry("version field is different",
				domain.MaintenanceInfo{Version: "1.2.0"},
				domain.MaintenanceInfo{Version: "2.2.2"},
			),
			Entry(
				"all properties are missing in one of the objects",
				domain.MaintenanceInfo{
					Public:      map[string]string{"foo": "bar"},
					Private:     "test",
					Version:     "1.2.3",
					Description: "test",
				},
				domain.MaintenanceInfo{}),
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
			Entry("description field is different",
				domain.MaintenanceInfo{Description: "amazing"},
				domain.MaintenanceInfo{Description: "terrible"},
			),
		)
	})
})
