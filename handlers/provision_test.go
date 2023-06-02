package handlers_test

import (
	"context"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v10/domain"
	"github.com/pivotal-cf/brokerapi/v10/domain/apiresponses"
	brokerFakes "github.com/pivotal-cf/brokerapi/v10/fakes"
	"github.com/pivotal-cf/brokerapi/v10/handlers"
	"github.com/pivotal-cf/brokerapi/v10/handlers/fakes"
	"github.com/pivotal-cf/brokerapi/v10/middlewares"
)

var _ = Describe("Provision", func() {
	var (
		fakeServiceBroker  *brokerFakes.AutoFakeServiceBroker
		fakeResponseWriter *fakes.FakeResponseWriter
		apiHandler         handlers.APIHandler
	)

	BeforeEach(func() {

		fakeServiceBroker = new(brokerFakes.AutoFakeServiceBroker)

		apiHandler = handlers.NewApiHandler(fakeServiceBroker, lager.NewLogger("test"))

		fakeResponseWriter = new(fakes.FakeResponseWriter)
		fakeResponseWriter.HeaderReturns(http.Header{})
	})

	It("can handle custom failure responses", func() {
		request := newProvisionRequest()

		expectedServices := []domain.Service{
			{
				ID:   "a-service",
				Name: "a-service",
				Plans: []domain.ServicePlan{
					{ID: "a-plan"},
				},
			},
		}

		fakeServiceBroker.ServicesReturns(expectedServices, nil)

		subFunctionWithCustomError := func() *apiresponses.FailureResponse {
			return nil
		}
		fakeServiceBroker.ProvisionStub = func(_ context.Context, _ string, _ domain.ProvisionDetails, _ bool) (domain.ProvisionedServiceSpec, error) {
			return domain.ProvisionedServiceSpec{}, subFunctionWithCustomError()
		}

		apiHandler.Provision(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusCreated))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).ToNot(BeEmpty())
	})

})

func newProvisionRequest() *http.Request {
	request, err := http.NewRequest(
		http.MethodGet,
		"https://broker.url/v2/service_instances/instance-id",
		strings.NewReader(`{"service_id":"a-service","plan_id":"a-plan"}`),
	)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Add("X-Broker-API-Version", "2.13")

	newCtx := context.WithValue(request.Context(), middlewares.CorrelationIDKey, "fake-correlation-id")
	request = request.WithContext(newCtx)
	return request
}
