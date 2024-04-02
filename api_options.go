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

	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/auth"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

type middlewareFunc func(http.Handler) http.Handler

type config struct {
	router               chi.Router
	customRouter         bool
	additionalMiddleware []middlewareFunc
}

func NewWithOptions(serviceBroker domain.ServiceBroker, logger *slog.Logger, opts ...Option) http.Handler {
	cfg := config{router: chi.NewRouter()}

	WithOptions(append(opts, withDefaultMiddleware(logger))...)(&cfg)
	attachRoutes(cfg.router, serviceBroker, logger)

	return cfg.router
}

type Option func(*config)

func WithRouter(router chi.Router) Option {
	return func(c *config) {
		c.router = router
		c.customRouter = true
	}
}

func WithBrokerCredentials(brokerCredentials BrokerCredentials) Option {
	return func(c *config) {
		c.router.Use(auth.NewWrapper(brokerCredentials.Username, brokerCredentials.Password).Wrap)
	}
}

// WithCustomAuth adds the specified middleware *before* any other middleware.
// Despite the name, any middleware can be added whether nor not it has anything to do with authentication.
// But `WithAdditionalMiddleware()` may be a better choice if the middleware is not related to authentication.
// Can be called multiple times.
func WithCustomAuth(authMiddleware middlewareFunc) Option {
	return func(c *config) {
		c.router.Use(authMiddleware)
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

// WithEncodedPath used to opt in to a gorilla/mux behaviour that would treat encoded
// slashes "/" as IDs. For example, it would change `PUT /v2/service_instances/foo%2Fbar`
// to treat `foo%2Fbar` as an instance ID, while the default behavior was to treat it
// as `foo/bar`. However, with moving to go-chi/chi, this is now the default behavior
// so this option no longer does anything.
//
// Deprecated: no longer has any effect
func WithEncodedPath() Option {
	return func(*config) {}
}

func withDefaultMiddleware(logger *slog.Logger) Option {
	return func(c *config) {
		if !c.customRouter {
			defaults := []middlewareFunc{
				middlewares.APIVersionMiddleware{Logger: logger}.ValidateAPIVersionHdr,
				middlewares.AddCorrelationIDToContext,
				middlewares.AddOriginatingIdentityToContext,
				middlewares.AddInfoLocationToContext,
				middlewares.AddRequestIdentityToContext,
			}

			for _, m := range append(defaults, c.additionalMiddleware...) {
				c.router.Use(m)
			}
		}
	}
}

func WithOptions(opts ...Option) Option {
	return func(c *config) {
		for _, o := range opts {
			o(c)
		}
	}
}
