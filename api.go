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
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v12/internal/middleware"

	"github.com/pivotal-cf/brokerapi/v12/auth"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/handlers"
	"github.com/pivotal-cf/brokerapi/v12/middlewares"
)

type BrokerCredentials struct {
	Username string
	Password string
}

func New(serviceBroker domain.ServiceBroker, logger *slog.Logger, brokerCredentials BrokerCredentials, opts ...Option) http.Handler {
	return NewWithOptions(serviceBroker, logger, append([]Option{WithBrokerCredentials(brokerCredentials)}, opts...)...)
}

func NewWithOptions(serviceBroker domain.ServiceBroker, logger *slog.Logger, opts ...Option) http.Handler {
	var cfg config
	WithOptions(opts...)(&cfg)

	mw := append(append(cfg.authMiddleware, defaultMiddleware(logger)...), cfg.additionalMiddleware...)
	r := router(serviceBroker, logger)

	return middleware.Use(r, mw...)
}

func NewWithCustomAuth(serviceBroker domain.ServiceBroker, logger *slog.Logger, authMiddleware func(handler http.Handler) http.Handler) http.Handler {
	return NewWithOptions(serviceBroker, logger, WithCustomAuth(authMiddleware))
}

type config struct {
	authMiddleware       []func(http.Handler) http.Handler
	additionalMiddleware []func(http.Handler) http.Handler
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
func WithCustomAuth(authMiddleware func(handler http.Handler) http.Handler) Option {
	return func(c *config) {
		c.authMiddleware = append(c.authMiddleware, authMiddleware)
	}
}

// WithAdditionalMiddleware adds the specified middleware *after* the default middleware.
// Can be called multiple times.
func WithAdditionalMiddleware(m func(http.Handler) http.Handler) Option {
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

	r.HandleFunc("PUT /v2/service_instances/{instance_id}", apiHandler.Provision)
	r.HandleFunc("GET /v2/service_instances/{instance_id}", apiHandler.GetInstance)
	r.HandleFunc("PATCH /v2/service_instances/{instance_id}", apiHandler.Update)
	r.HandleFunc("DELETE /v2/service_instances/{instance_id}", apiHandler.Deprovision)

	r.HandleFunc("GET /v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation)

	r.HandleFunc("PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind)
	r.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding)
	r.HandleFunc("DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind)

	r.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation)

	return r
}

func defaultMiddleware(logger *slog.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middlewares.APIVersionMiddleware{Logger: logger}.ValidateAPIVersionHdr,
		middlewares.AddCorrelationIDToContext,
		middlewares.AddOriginatingIdentityToContext,
		middlewares.AddInfoLocationToContext,
		middlewares.AddRequestIdentityToContext,
	}
}
