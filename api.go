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

package brokerapi

import (
	"github.com/pivotal-cf/brokerapi/v11/auth"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/handlers"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
	"log/slog"
	"net/http"
	"slices"
)

type BrokerCredentials struct {
	Username string
	Password string
}

type middlewareFunc func(http.Handler) http.Handler

type config struct {
	authMiddleware       []middlewareFunc
	additionalMiddleware []middlewareFunc
}

func New(serviceBroker domain.ServiceBroker, logger *slog.Logger, opts ...Option) http.Handler {
	var cfg config
	WithOptions(opts...)(&cfg)

	mw := combineMiddlewares(append(append(cfg.authMiddleware, defaultMiddleware(logger)...), cfg.additionalMiddleware...)...)
	r := router(serviceBroker, logger)

	return mw(r)
}

type Option func(*config)

func WithBrokerCredentials(brokerCredentials BrokerCredentials) Option {
	return func(c *config) {
		c.authMiddleware = append(c.authMiddleware, auth.NewWrapper(brokerCredentials.Username, brokerCredentials.Password).Wrap)
	}
}

// WithCustomAuth adds the specified middleware *before* any other middleware.
// Despite the name, any middleware can be added whether nor not it has anything to do with authentication.
// But `WithAdditionalMiddleware()` may be a better choice if the middleware is not related to authentication.
// Can be called multiple times.
func WithCustomAuth(authMiddleware middlewareFunc) Option {
	return func(c *config) {
		c.authMiddleware = append(c.authMiddleware, authMiddleware)
	}
}

// WithAdditionalMiddleware adds the specified middleware *after* the default middleware.
// Can be called multiple times.
// This option is ignored if `WithRouter()` is used.
func WithAdditionalMiddleware(m middlewareFunc) Option {
	return func(c *config) {
		c.additionalMiddleware = append(c.additionalMiddleware, m)
	}
}

func WithOptions(opts ...Option) Option {
	return func(c *config) {
		for _, o := range opts {
			o(c)
		}
	}
}

func router(serviceBroker ServiceBroker, logger *slog.Logger) http.Handler {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	r := http.NewServeMux()
	r.HandleFunc("GET /v2/catalog", apiHandler.Catalog)

	r.HandleFunc("GET /v2/service_instances/{instance_id}", apiHandler.GetInstance)
	r.HandleFunc("PUT /v2/service_instances/{instance_id}", apiHandler.Provision)
	r.HandleFunc("DELETE /v2/service_instances/{instance_id}", apiHandler.Deprovision)
	r.HandleFunc("GET /v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation)
	r.HandleFunc("PATCH /v2/service_instances/{instance_id}", apiHandler.Update)

	r.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding)
	r.HandleFunc("PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind)
	r.HandleFunc("DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind)

	r.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation)

	return r
}

func defaultMiddleware(logger *slog.Logger) []middlewareFunc {
	return []middlewareFunc{
		middlewares.APIVersionMiddleware{Logger: logger}.ValidateAPIVersionHdr,
		middlewares.AddCorrelationIDToContext,
		middlewares.AddOriginatingIdentityToContext,
		middlewares.AddInfoLocationToContext,
		middlewares.AddRequestIdentityToContext,
	}
}

func combineMiddlewares(middlewares ...middlewareFunc) middlewareFunc {
	sorted := sortMiddlewares(middlewares)
	return func(next http.Handler) http.Handler {
		for _, m := range sorted {
			next = m(next)
		}
		return next
	}
}

func sortMiddlewares(middlewares []middlewareFunc) (result []middlewareFunc) {
	slices.Reverse(middlewares)
	for _, m := range middlewares {
		if m != nil {
			result = append(result, m)
		}
	}
	return result
}
