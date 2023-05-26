package brokerapi_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/brokerapi/v10"
)

var _ = Describe("Context Utilities", func() {

	type testContextKey string

	var (
		ctx                   context.Context
		contextValidatorKey   testContextKey
		contextValidatorValue string
	)

	BeforeEach(func() {
		contextValidatorKey = "context-utilities-test"
		contextValidatorValue = "original"
		ctx = context.Background()
		ctx = context.WithValue(ctx, contextValidatorKey, contextValidatorValue)
	})

	Describe("Service Context", func() {
		Context("when the service is nil", func() {
			It("returns the original context", func() {
				ctx = brokerapi.AddServiceToContext(ctx, nil)
				Expect(ctx.Err()).To(BeZero())
				Expect(brokerapi.RetrieveServiceFromContext(ctx)).To(BeZero())
				Expect(ctx.Value(contextValidatorKey).(string)).To(Equal(contextValidatorValue))
			})
		})

		Context("when the service is valid", func() {
			It("sets and receives the service in the context", func() {
				service := &brokerapi.Service{
					ID:   "9A3095D7-ED3C-45FA-BC9F-592820628723",
					Name: "Test Service",
				}
				ctx = brokerapi.AddServiceToContext(ctx, service)
				Expect(ctx.Err()).To(BeZero())
				Expect(ctx.Value(contextValidatorKey).(string)).To(Equal(contextValidatorValue))
				Expect(brokerapi.RetrieveServiceFromContext(ctx).ID).To(Equal(service.ID))
				Expect(brokerapi.RetrieveServiceFromContext(ctx).Name).To(Equal(service.Name))
				Expect(brokerapi.RetrieveServiceFromContext(ctx).Metadata).To(BeZero())
			})
		})
	})

	Describe("Plan Context", func() {
		Context("when the service plan is nil", func() {
			It("returns the original context", func() {
				ctx = brokerapi.AddServicePlanToContext(ctx, nil)
				Expect(ctx.Err()).To(BeZero())
				Expect(brokerapi.RetrieveServicePlanFromContext(ctx)).To(BeZero())
				Expect(ctx.Value(contextValidatorKey).(string)).To(Equal(contextValidatorValue))
			})
		})

		Context("when the service plan is valid", func() {
			It("sets and retrieves the service plan in the context", func() {
				plan := &brokerapi.ServicePlan{
					ID: "AC257573-8C62-4B1A-AC34-ECA3863F50EC",
				}
				ctx = brokerapi.AddServicePlanToContext(ctx, plan)
				Expect(ctx.Err()).To(BeZero())
				Expect(ctx.Value(contextValidatorKey).(string)).To(Equal(contextValidatorValue))
				Expect(brokerapi.RetrieveServicePlanFromContext(ctx).ID).To(Equal(plan.ID))
			})
		})
	})
})
