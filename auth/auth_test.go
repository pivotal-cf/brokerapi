package auth_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/brokerapi/auth"
)

var _ = Describe("Auth Wrapper", func() {
	var (
		wrappedHandler http.Handler
		username       string
		password       string
	)

	BeforeEach(func() {
		username = "username"
		password = "password"

		authWrapper := auth.NewWrapper(username, password)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
		wrappedHandler = authWrapper.Wrap(handler)
	})

	It("works when the credentials are correct", func() {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, password)

		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusCreated))
	})

	It("fails when the username is empty", func() {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth("", password)

		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("fails when the password is empty", func() {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(username, "")

		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("fails when the credentials are wrong", func() {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth("thats", "apar")

		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})
})
