// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package brokerapi

import (
	"net/http"

	"fmt"

	"code.cloudfoundry.org/lager"
)

type FailureResponse struct {
	error
	statusCode    int
	loggerAction  string
	emptyResponse bool
	errorKey      string
}

type FailureResponseBuilder struct {
	error
	statusCode    int
	loggerAction  string
	emptyResponse bool
	errorKey      string
}

func (f *FailureResponseBuilder) WithErrorKey(errorKey string) *FailureResponseBuilder {
	f.errorKey = errorKey
	return f
}

func (f *FailureResponseBuilder) WithEmptyResponse() *FailureResponseBuilder {
	f.emptyResponse = true
	return f
}

func (f *FailureResponseBuilder) Build() *FailureResponse {
	return &FailureResponse{
		error:         f.error,
		statusCode:    f.statusCode,
		loggerAction:  f.loggerAction,
		emptyResponse: f.emptyResponse,
		errorKey:      f.errorKey,
	}
}

func NewFailureResponseBuilder(err error, statusCode int, loggerAction string) *FailureResponseBuilder {
	return &FailureResponseBuilder{
		error:         err,
		statusCode:    statusCode,
		loggerAction:  loggerAction,
		emptyResponse: false,
	}
}

func NewFailureResponse(err error, statusCode int, loggerAction string) *FailureResponse {
	return &FailureResponse{
		error:        err,
		statusCode:   statusCode,
		loggerAction: loggerAction,
	}
}

func (f *FailureResponse) ErrorResponse() interface{} {
	if f.emptyResponse {
		return EmptyResponse{}
	}

	return ErrorResponse{
		Description: f.error.Error(),
		Error:       f.errorKey,
	}
}

func (f *FailureResponse) ValidatedStatusCode(logger lager.Logger) int {
	if f.statusCode < 400 || 600 <= f.statusCode {
		if logger != nil {
			logger.Error("validating-status-code", fmt.Errorf("Invalid failure http response code: 600, expected 4xx or 5xx, returning internal server error: 500."))
		}
		return http.StatusInternalServerError
	}
	return f.statusCode
}

func (f *FailureResponse) LoggerAction() string {
	return f.loggerAction
}
