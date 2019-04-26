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
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
				},
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				}),
			Entry(
				"one property is different",
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test-not-the-same",
					Version: "1.2.3",
				}),
			Entry(
				"all properties are missing in one of the objects",
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				domain.MaintenanceInfo{}),
			Entry(
				"all properties are defined but different",
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				domain.MaintenanceInfo{
					Public:  map[string]string{"bar": "foo"},
					Private: "test-not-the-same",
					Version: "8.9.6-rc3",
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
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
				},
				domain.MaintenanceInfo{
					Public:  map[string]string{"foo": "bar"},
					Private: "test",
					Version: "1.2.3",
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

	Describe("NilOrEmpty", func() {
		It("return true when maintenance_info is nil", func() {
			var m *domain.MaintenanceInfo = nil

			Expect(m.NilOrEmpty()).To(BeTrue())
		})

		It("return true when maintenance_info is empty", func() {
			var m = &domain.MaintenanceInfo{
				Public:  nil,
				Private: "",
				Version: "",
			}

			Expect(m.NilOrEmpty()).To(BeTrue())
		})

		It("return false when maintenance_info has properties", func() {
			m := &domain.MaintenanceInfo{
				Public: map[string]string{
					"test": "foo",
				},
				Private: "test-again",
				Version: "1.2.3",
			}

			Expect(m.NilOrEmpty()).To(BeFalse())
		})
	})

})
