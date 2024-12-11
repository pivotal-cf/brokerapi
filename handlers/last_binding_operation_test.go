package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v11"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	brokerFakes "github.com/pivotal-cf/brokerapi/v11/fakes"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

var _ = Describe("LastBindingOperation", func() {
	const (
		instanceID = "some-instance-id"
		bindingID  = "some-binding-id"
		planID     = "a-plan"
		serviceID  = "a-service"
		operation  = "a-operation"
	)

	var (
		fakeServiceBroker *brokerFakes.AutoFakeServiceBroker
		fakeServer        *httptest.Server
	)

	BeforeEach(func() {
		fakeServiceBroker = new(brokerFakes.AutoFakeServiceBroker)
		fakeServer = httptest.NewServer(brokerapi.NewWithOptions(fakeServiceBroker, slog.New(slog.NewJSONHandler(GinkgoWriter, nil))))
	})

	It("responds with OK when broker can retrieve the last binding operation", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation, fakeServer.URL)
		expectedLastOperation := domain.LastOperation{
			State:       domain.Succeeded,
			Description: "muy bien",
		}

		fakeServiceBroker.LastBindingOperationReturns(expectedLastOperation, nil)

		response := must(fakeServer.Client().Do(request))
		Expect(response).To(HaveHTTPStatus(http.StatusOK))
		Expect(readBody(response)).To(MatchJSON(toJSON(expectedLastOperation)))

		_, actualInstanceID, actualBindingID, actualPollDetails := fakeServiceBroker.LastBindingOperationArgsForCall(0)
		Expect(actualPollDetails).To(Equal(domain.PollDetails{
			PlanID:        planID,
			ServiceID:     serviceID,
			OperationData: operation,
		}))
		Expect(actualInstanceID).To(Equal(instanceID))
		Expect(actualBindingID).To(Equal(bindingID))
	})

	It("responds with PreConditionFailed when api version is not supported", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation, fakeServer.URL)
		request.Header.Set("X-Broker-API-Version", "2.13")

		response := must(fakeServer.Client().Do(request))
		Expect(response).To(HaveHTTPStatus(http.StatusPreconditionFailed))
		Expect(readBody(response)).To(MatchJSON(`{"description":"get binding endpoint only supported starting with OSB version 2.14"}`))
	})

	It("responds with InternalServerError when last binding operation returns unknown error", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation, fakeServer.URL)

		fakeServiceBroker.LastBindingOperationReturns(domain.LastOperation{}, errors.New("some error"))

		response := must(fakeServer.Client().Do(request))
		Expect(response).To(HaveHTTPStatus(http.StatusInternalServerError))
		Expect(readBody(response)).To(MatchJSON(`{"description":"some error"}`))
	})

	It("responds appropriately when last binding operation returns a known error", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation, fakeServer.URL)
		err := errors.New("some-amazing-error")
		fakeServiceBroker.LastBindingOperationReturns(
			domain.LastOperation{},
			apiresponses.NewFailureResponse(err, http.StatusTeapot, "last-binding-op"),
		)

		response := must(fakeServer.Client().Do(request))
		Expect(response).To(HaveHTTPStatus(http.StatusTeapot))
		Expect(readBody(response)).To(MatchJSON(`{"description":"some-amazing-error"}`))
	})
})

func toJSON(operation domain.LastOperation) []byte {
	d, err := json.Marshal(operation)
	Expect(err).ToNot(HaveOccurred())
	return d
}

func newRequest(instanceID, bindingID, planID, serviceID, operation, serverURL string) *http.Request {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s/last_operation", serverURL, instanceID, bindingID),
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Add("X-Broker-API-Version", "2.14")

	q := request.URL.Query()
	q.Add("plan_id", planID)
	q.Add("service_id", serviceID)
	q.Add("operation", operation)
	request.URL.RawQuery = q.Encode()

	ctx := request.Context()
	ctx = context.WithValue(ctx, middlewares.CorrelationIDKey, "fake-correlation-id")

	return request.WithContext(ctx)
}

func readBody(res *http.Response) string {
	GinkgoHelper()

	body := must(io.ReadAll(res.Body))
	res.Body.Close()
	return string(body)
}

func must[A any](input A, err error) A {
	GinkgoHelper()

	Expect(err).ToNot(HaveOccurred())
	return input
}
