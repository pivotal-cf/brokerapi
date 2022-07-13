package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pivotal-cf/brokerapi/v8/middlewares"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	brokerFakes "github.com/pivotal-cf/brokerapi/v8/fakes"
	"github.com/pivotal-cf/brokerapi/v8/handlers"
	"github.com/pivotal-cf/brokerapi/v8/handlers/fakes"
	"github.com/pkg/errors"
)

var _ = Describe("LastBindingOperation", func() {
	var (
		fakeServiceBroker  *brokerFakes.AutoFakeServiceBroker
		fakeResponseWriter *fakes.FakeResponseWriter
		apiHandler         handlers.APIHandler

		instanceID, bindingID        string
		planID, serviceID, operation string
	)

	BeforeEach(func() {
		instanceID = "some-instance-id"
		bindingID = "some-binding-id"
		planID = "a-plan"
		serviceID = "a-service"
		operation = "a-operation"

		fakeServiceBroker = new(brokerFakes.AutoFakeServiceBroker)

		apiHandler = handlers.NewApiHandler(fakeServiceBroker, lager.NewLogger("test"))

		fakeResponseWriter = new(fakes.FakeResponseWriter)
		fakeResponseWriter.HeaderReturns(http.Header{})
	})

	It("responds with OK when broker can retrieve the last binding operation", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation)
		expectedLastOperation := domain.LastOperation{
			State:       domain.Succeeded,
			Description: "muy bien",
		}

		fakeServiceBroker.LastBindingOperationReturns(expectedLastOperation, nil)

		apiHandler.LastBindingOperation(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusOK))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(toJSON(expectedLastOperation)))

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
		request := newRequest(instanceID, bindingID, planID, serviceID, operation)
		request.Header.Set("X-Broker-API-Version", "2.13")

		apiHandler.LastBindingOperation(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusPreconditionFailed))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(`{"description":"get binding endpoint only supported starting with OSB version 2.14"}`))
	})

	It("responds with InternalServerError when last binding operation returns unknown error", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation)

		fakeServiceBroker.LastBindingOperationReturns(domain.LastOperation{}, errors.New("some error"))

		apiHandler.LastBindingOperation(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusInternalServerError))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(`{"description":"some error"}`))
	})

	It("responds appropriately when last binding operation returns a known error", func() {
		request := newRequest(instanceID, bindingID, planID, serviceID, operation)
		err := errors.New("some-amazing-error")
		fakeServiceBroker.LastBindingOperationReturns(
			domain.LastOperation{},
			apiresponses.NewFailureResponse(err, http.StatusTeapot, "last-binding-op"),
		)

		apiHandler.LastBindingOperation(fakeResponseWriter, request)

		statusCode := fakeResponseWriter.WriteHeaderArgsForCall(0)
		Expect(statusCode).To(Equal(http.StatusTeapot))
		body := fakeResponseWriter.WriteArgsForCall(0)
		Expect(body).To(MatchJSON(`{"description":"some-amazing-error"}`))
	})
})

func toJSON(operation domain.LastOperation) []byte {
	d, err := json.Marshal(operation)
	Expect(err).ToNot(HaveOccurred())
	return d
}

func newRequest(instanceID, bindingID, planID, serviceID, operation string) *http.Request {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://broker.url/v2/service_instances/%s/service_bindings/%s/last_operation", instanceID, bindingID),
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	request.Header.Add("X-Broker-API-Version", "2.14")

	request = mux.SetURLVars(request, map[string]string{
		"instance_id": instanceID,
		"binding_id":  bindingID,
	})

	request.Form = url.Values{}
	request.Form.Add("plan_id", planID)
	request.Form.Add("service_id", serviceID)
	request.Form.Add("operation", operation)

	newCtx := context.WithValue(request.Context(), middlewares.CorrelationIDKey, "fake-correlation-id")
	request = request.WithContext(newCtx)
	return request
}
