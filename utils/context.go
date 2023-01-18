package utils

import (
	"context"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/brokerapi/v9/domain"
	"github.com/pivotal-cf/brokerapi/v9/middlewares"
)

type contextKey string

const (
	contextKeyService contextKey = "brokerapi_service"
	contextKeyPlan    contextKey = "brokerapi_plan"
)

func AddServiceToContext(ctx context.Context, service *domain.Service) context.Context {
	if service != nil {
		return context.WithValue(ctx, contextKeyService, service)
	}
	return ctx
}

func RetrieveServiceFromContext(ctx context.Context) *domain.Service {
	if value := ctx.Value(contextKeyService); value != nil {
		return value.(*domain.Service)
	}
	return nil
}

func AddServicePlanToContext(ctx context.Context, plan *domain.ServicePlan) context.Context {
	if plan != nil {
		return context.WithValue(ctx, contextKeyPlan, plan)
	}
	return ctx
}

func RetrieveServicePlanFromContext(ctx context.Context) *domain.ServicePlan {
	if value := ctx.Value(contextKeyPlan); value != nil {
		return value.(*domain.ServicePlan)
	}
	return nil
}

func DataForContext(context context.Context, dataKeys ...middlewares.ContextKey) lager.Data {
	data := lager.Data{}
	for _, key := range dataKeys {
		if value := context.Value(key); value != nil {
			data[string(key)] = value
		}
	}

	return data
}
