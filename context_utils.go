package brokerapi

import "context"

type contextKey string

const (
	contextKeyService contextKey = "brokerapi_service"
	contextKeyPlan    contextKey = "brokerapi_plan"
)

func AddServiceToContext(ctx context.Context, service *Service) context.Context {
	if service != nil {
		return context.WithValue(ctx, contextKeyService, service)
	}
	return ctx
}

func RetrieveServiceFromContext(ctx context.Context) *Service {
	if value := ctx.Value(contextKeyService); value != nil {
		return value.(*Service)
	}
	return nil
}

func AddServicePlanToContext(ctx context.Context, plan *ServicePlan) context.Context {
	if plan != nil {
		return context.WithValue(ctx, contextKeyPlan, plan)
	}
	return ctx
}

func RetrieveServicePlanFromContext(ctx context.Context) *ServicePlan {
	if value := ctx.Value(contextKeyPlan); value != nil {
		return value.(*ServicePlan)
	}
	return nil
}
