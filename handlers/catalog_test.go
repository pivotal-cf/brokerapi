package handlers_test

import (
	"context"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	brokerFakes "github.com/pivotal-cf/brokerapi/v10/fakes"
	"github.com/pivotal-cf/brokerapi/v10/handlers"
	"github.com/pivotal-cf/brokerapi/v10/handlers/fakes"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
)

var _ = Describe("Services", func() {
	var (
		fakeServiceBroker  *brokerFakes.AutoFakeServiceBroker
		fakeResponseWriter *fakes.FakeResponseWriter
		apiHandler         handlers.APIHandler

		serviceID string
	)

	BeforeEach(func() {
		serviceID = "a-service"

		fakeServiceBroker = new(brokerFakes.AutoFakeServiceBroker)

		apiHandler = handlers.NewApiHandler(fakeServiceBroker, lager.NewLogger("test"))

		fakeResponseWriter = new(fakes.FakeResponseWriter)
		fakeResponseWriter.HeaderReturns(http.Header{})
	})

	It("responds with OK when broker can retrieve the services catalog", func() {
		request := newServicesRequest()
		expectedServices := []domain.Service{
			{
				ID:          serviceID,
				Name:        serviceID,
				Description: "muy bien",
			},
		}

		fakeServiceBroker.ServicesReturns(expectedServices, nil)

		apiHandler.Catalog(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusOK))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).ToNot(BeEmpty())
	})

	It("responds with InternalServerError when services catalog returns unknown error", func() {
		request := newServicesRequest()

		fakeServiceBroker.ServicesReturns(nil, errors.New("some error"))

		apiHandler.Catalog(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusInternalServerError))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(`{"description":"some error"}`))
	})

	It("responds with status code set in the FailureResponse when services catalog returns it", func() {
		request := newServicesRequest()

		fakeServiceBroker.ServicesReturns(
			nil,
			apiresponses.NewFailureResponse(
				errors.New("TODO"),
				http.StatusNotImplemented,
				http.StatusText(http.StatusNotImplemented),
			),
		)

		apiHandler.Catalog(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusNotImplemented))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(`{"description":"TODO"}`))
	})
})

func newServicesRequest() *http.Request {
	request, err := http.NewRequest(
		"GET",
		"https://broker.url/v2/catalog",
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Add("X-Broker-API-Version", "2.13")

	newCtx := context.WithValue(request.Context(), middlewares.CorrelationIDKey, "fake-correlation-id")
	request = request.WithContext(newCtx)
	return request
}
