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
	"github.com/pivotal-cf/brokerapi/v11/handlers"
)

type BrokerCredentials struct {
	Username string
	Password string
}

func New(serviceBroker ServiceBroker, logger *slog.Logger, brokerCredentials BrokerCredentials) http.Handler {
	return NewWithOptions(serviceBroker, logger, WithBrokerCredentials(brokerCredentials))
}

func NewWithCustomAuth(serviceBroker ServiceBroker, logger *slog.Logger, authMiddleware middlewareFunc) http.Handler {
	return NewWithOptions(serviceBroker, logger, WithCustomAuth(authMiddleware))
}

func AttachRoutes(router chi.Router, serviceBroker ServiceBroker, logger *slog.Logger) {
	attachRoutes(router, serviceBroker, logger)
}

func attachRoutes(router chi.Router, serviceBroker ServiceBroker, logger *slog.Logger) {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	router.Get("/v2/catalog", apiHandler.Catalog)

	router.Get("/v2/service_instances/{instance_id}", apiHandler.GetInstance)
	router.Put("/v2/service_instances/{instance_id}", apiHandler.Provision)
	router.Delete("/v2/service_instances/{instance_id}", apiHandler.Deprovision)
	router.Get("/v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation)
	router.Patch("/v2/service_instances/{instance_id}", apiHandler.Update)

	router.Get("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding)
	router.Put("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind)
	router.Delete("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind)

	router.Get("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation)
}
