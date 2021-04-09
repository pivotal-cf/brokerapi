package middlewares

import (
	"context"
	"net/http"
)

func AddRequestIdentityToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestIdentity := req.Header.Get("X-Broker-API-Request-Identity")
		newCtx := context.WithValue(req.Context(), RequestIdentityKey, requestIdentity)
		next.ServeHTTP(w, req.WithContext(newCtx))
	})
}
