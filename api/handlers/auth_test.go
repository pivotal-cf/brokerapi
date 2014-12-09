package handlers_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/go-service-broker/api/handlers"
)

var lastWrittenStatusCode int

type TestResponseWriter struct{}

func (TestResponseWriter) Header() http.Header {
	return http.Header{}
}

func (TestResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (TestResponseWriter) WriteHeader(statusCode int) {
	lastWrittenStatusCode = statusCode
}

var _ = Describe("http basic auth", func() {
	It("works when the credentials are correct", func() {
		var username = "some_username"
		var password = "some_password"

		lastWrittenStatusCode = http.StatusOK
		handler := handlers.CheckAuth(username, password)

		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, password)

		handler(TestResponseWriter{}, request)

		Expect(lastWrittenStatusCode).To(Equal(http.StatusOK))
	})

	It("fails when the username is empty", func() {
		var username = ""
		var password = "password"

		lastWrittenStatusCode = http.StatusOK
		handler := handlers.CheckAuth(username, password)

		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, password)

		handler(TestResponseWriter{}, request)

		Expect(lastWrittenStatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("fails when the password is empty", func() {
		var username = "some_username"
		var password = ""

		lastWrittenStatusCode = http.StatusOK
		handler := handlers.CheckAuth(username, password)

		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, password)

		handler(TestResponseWriter{}, request)

		Expect(lastWrittenStatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("fails when the credentails are incorrect", func() {
		var username = "some_username"
		var password = "some_password"

		lastWrittenStatusCode = http.StatusOK
		handler := handlers.CheckAuth(username, password)

		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, "wrong_password")

		handler(TestResponseWriter{}, request)

		Expect(lastWrittenStatusCode).To(Equal(http.StatusUnauthorized))
	})
})
