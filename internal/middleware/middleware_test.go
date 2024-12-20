package middleware_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v11/internal/middleware"
)

var _ = Describe("Use()", func() {
	It("can handle an empty list", func() {
		endpointCalled := false
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			endpointCalled = true
		})

		middleware.Use(endpoint).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

		Expect(endpointCalled).To(BeTrue())
	})

	It("calls middleware in the right order", func() {
		var order []string
		first := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, "first")
				next.ServeHTTP(w, req)
			})
		}

		second := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, "second")
				next.ServeHTTP(w, req)
			})
		}

		third := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, "third")
				next.ServeHTTP(w, req)
			})
		}

		endpointCalled := false
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			endpointCalled = true
		})
		middleware.Use(endpoint, first, second, third).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

		Expect(order).To(Equal([]string{"first", "second", "third"}))
		Expect(endpointCalled).To(BeTrue())
	})

	It("does not call the endpoint when the middleware rejects the request", func() {
		rejector := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				http.Error(w, http.StatusText(http.StatusTeapot), http.StatusTeapot)
			})
		}

		endpointCalled := false
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			endpointCalled = true
		})
		middleware.Use(endpoint, rejector).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

		Expect(endpointCalled).To(BeFalse())
	})
})
