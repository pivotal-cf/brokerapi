package blog_test

import (
	"context"
	"encoding/json"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi/v11/internal/blog"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

var _ = Describe("Context data", func() {
	When("the context has the values", func() {
		It("logs the values", func() {
			const (
				correlationID = "fake-correlation-id"
				requestID     = "fake-request-id"
			)

			ctx := context.TODO()
			ctx = context.WithValue(ctx, middlewares.CorrelationIDKey, correlationID)
			ctx = context.WithValue(ctx, middlewares.RequestIdentityKey, requestID)

			buffer := gbytes.NewBuffer()
			logger := slog.New(slog.NewJSONHandler(buffer, nil))

			blog.New(logger).Session(ctx, "prefix").Info("hello")

			var receiver map[string]any
			Expect(json.Unmarshal(buffer.Contents(), &receiver)).To(Succeed())

			Expect(receiver).To(HaveKeyWithValue(string(middlewares.CorrelationIDKey), correlationID))
			Expect(receiver).To(HaveKeyWithValue(string(middlewares.RequestIdentityKey), requestID))
		})
	})

	When("the context does not have the values", func() {
		It("does not log them", func() {
			buffer := gbytes.NewBuffer()
			logger := slog.New(slog.NewJSONHandler(buffer, nil))

			blog.New(logger).Session(context.TODO(), "prefix").Info("hello")

			var receiver map[string]any
			Expect(json.Unmarshal(buffer.Contents(), &receiver)).To(Succeed())

			Expect(receiver).NotTo(HaveKey(string(middlewares.CorrelationIDKey)))
			Expect(receiver).NotTo(HaveKey(string(middlewares.RequestIdentityKey)))
		})
	})
})
