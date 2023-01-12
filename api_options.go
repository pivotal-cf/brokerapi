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
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v9/auth"
	"github.com/pivotal-cf/brokerapi/v9/domain"
	"github.com/pivotal-cf/brokerapi/v9/middlewares"
)

func NewWithOptions(serviceBroker domain.ServiceBroker, logger lager.Logger, opts ...Option) http.Handler {
	cfg := newDefaultConfig(logger)
	WithOptions(append(opts, withDefaultMiddleware())...)(cfg)
	attachRoutes(cfg.router, serviceBroker, logger)

	return cfg.router
}

type Option func(*config)

func WithRouter(router *mux.Router) Option {
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

func WithCustomAuth(authMiddleware mux.MiddlewareFunc) Option {
	return func(c *config) {
		c.router.Use(authMiddleware)
	}
}

func WithEncodedPath() Option {
	return func(c *config) {
		c.router.UseEncodedPath()
	}
}

func withDefaultMiddleware() Option {
	return func(c *config) {
		if !c.customRouter {
			c.router.Use(middlewares.APIVersionMiddleware{LoggerFactory: c.logger}.ValidateAPIVersionHdr)
			c.router.Use(middlewares.AddCorrelationIDToContext)
			c.router.Use(middlewares.AddOriginatingIdentityToContext)
			c.router.Use(middlewares.AddInfoLocationToContext)
			c.router.Use(middlewares.AddRequestIdentityToContext)
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

func newDefaultConfig(logger lager.Logger) *config {
	return &config{
		router:       mux.NewRouter(),
		customRouter: false,
		logger:       logger,
	}
}

type config struct {
	router       *mux.Router
	customRouter bool
	logger       lager.Logger
}
