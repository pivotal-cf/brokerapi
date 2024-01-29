package apiresponses_test

import (
	"errors"
	"log/slog"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
)

var _ = Describe("FailureResponse", func() {
	Describe("ErrorResponse", func() {
		It("returns a ErrorResponse containing the error message", func() {
			failureResponse := asFailureResponse(apiresponses.NewFailureResponse(errors.New("my error message"), http.StatusForbidden, "log-key"))
			Expect(failureResponse.ErrorResponse()).To(Equal(apiresponses.ErrorResponse{
				Description: "my error message",
			}))
		})

		Context("when the error key is provided", func() {
			It("returns a ErrorResponse containing the error message and the error key", func() {
				failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithErrorKey("error key").Build()
				Expect(failureResponse.ErrorResponse()).To(Equal(apiresponses.ErrorResponse{
					Description: "my error message",
					Error:       "error key",
				}))
			})
		})

		Context("when created with empty response", func() {
			It("returns an EmptyResponse", func() {
				failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithEmptyResponse().Build()
				Expect(failureResponse.ErrorResponse()).To(Equal(apiresponses.EmptyResponse{}))
			})
		})
	})

	Describe("AppendErrorMessage", func() {
		It("returns the error with the additional error message included, with a non-empty body", func() {
			failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithErrorKey("some-key").Build()
			Expect(failureResponse.Error()).To(Equal("my error message"))

			newError := failureResponse.AppendErrorMessage("and some more details")

			Expect(newError.Error()).To(Equal("my error message and some more details"))
			Expect(newError.ValidatedStatusCode("", nil)).To(Equal(http.StatusForbidden))
			Expect(newError.LoggerAction()).To(Equal(failureResponse.LoggerAction()))

			errorResponse, typeCast := newError.ErrorResponse().(apiresponses.ErrorResponse)
			Expect(typeCast).To(BeTrue())
			Expect(errorResponse.Error).To(Equal("some-key"))
			Expect(errorResponse.Description).To(Equal("my error message and some more details"))
		})

		It("returns the error with the additional error message included, with an empty body", func() {
			failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithEmptyResponse().Build()
			Expect(failureResponse.Error()).To(Equal("my error message"))

			newError := failureResponse.AppendErrorMessage("and some more details")

			Expect(newError.Error()).To(Equal("my error message and some more details"))
			Expect(newError.ValidatedStatusCode("", nil)).To(Equal(http.StatusForbidden))
			Expect(newError.LoggerAction()).To(Equal(failureResponse.LoggerAction()))
			Expect(newError.ErrorResponse()).To(Equal(failureResponse.ErrorResponse()))
		})
	})

	Describe("ValidatedStatusCode", func() {
		It("returns the status code that was passed in", func() {
			failureResponse := asFailureResponse(apiresponses.NewFailureResponse(errors.New("my error message"), http.StatusForbidden, "log-key"))
			Expect(failureResponse.ValidatedStatusCode("", nil)).To(Equal(http.StatusForbidden))
		})

		It("when error key is provided it returns the status code that was passed in", func() {
			failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithErrorKey("error key").Build()
			Expect(failureResponse.ValidatedStatusCode("", nil)).To(Equal(http.StatusForbidden))
		})

		Context("when the status code is invalid", func() {
			It("returns 500", func() {
				failureResponse := asFailureResponse(apiresponses.NewFailureResponse(errors.New("my error message"), 600, "log-key"))
				Expect(failureResponse.ValidatedStatusCode("", nil)).To(Equal(http.StatusInternalServerError))
			})

			It("logs that the status has been changed", func() {
				log := gbytes.NewBuffer()
				logger := slog.New(slog.NewJSONHandler(log, nil))
				failureResponse := asFailureResponse(apiresponses.NewFailureResponse(errors.New("my error message"), 600, "log-key"))
				failureResponse.ValidatedStatusCode("", logger)
				Expect(log).To(gbytes.Say("Invalid failure http response code: 600, expected 4xx or 5xx, returning internal server error: 500."))
			})
		})
	})

	Describe("LoggerAction", func() {
		It("returns the logger action that was passed in", func() {
			failureResponse := apiresponses.NewFailureResponseBuilder(errors.New("my error message"), http.StatusForbidden, "log-key").WithErrorKey("error key").Build()
			Expect(failureResponse.LoggerAction()).To(Equal("log-key"))
		})

		It("when error key is provided it returns the logger action that was passed in", func() {
			failureResponse := asFailureResponse(apiresponses.NewFailureResponse(errors.New("my error message"), http.StatusForbidden, "log-key"))
			Expect(failureResponse.LoggerAction()).To(Equal("log-key"))
		})
	})
})

func asFailureResponse(err error) *apiresponses.FailureResponse {
	GinkgoHelper()
	Expect(err).To(BeAssignableToTypeOf(&apiresponses.FailureResponse{}))
	return err.(*apiresponses.FailureResponse)
}
