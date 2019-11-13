package middlewares

import (
	"context"
	"net/http"

	"github.com/pborman/uuid"
)

const CorrelationIDKey = "correlation-id"

var correlationIDHeaders = []string{"X-Correlation-ID", "X-CorrelationID", "X-ForRequest-ID", "X-Request-ID", "X-Vcap-Request-Id"}

func AddCorrelationIDToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var correlationID string
		var found bool

		for _, header := range correlationIDHeaders {
			headerValue := req.Header.Get(header)
			if headerValue != "" {
				correlationID = headerValue
				found = true
				break
			}
		}

		if !found {
			correlationID = uuid.New()
		}

		newCtx := context.WithValue(req.Context(), CorrelationIDKey, correlationID)
		next.ServeHTTP(w, req.WithContext(newCtx))
	})
}
