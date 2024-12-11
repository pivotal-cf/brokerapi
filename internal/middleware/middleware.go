// Package middleware implements a middleware combiner. For implementations of middleware, see
// the middlewares package
package middleware

import (
	"net/http"
	"slices"
)

// Use will combine multiple middleware functions and apply them to the specified endpoint. Middleware is defined as:
//
//	func(next http.Handler) http.Handler
//
// Middleware is called (by calling ServeHTTP() on it, or as a http.HandlerFunc) and it should either:
// - respond via the http.ResponseWriter, typically with an error
// - call the next middleware
// Optionally it may also do something useful, such as modifying the context.
// Eventually, once all the middleware has been called, the endpoint will be called
func Use(endpoint http.Handler, middlewares ...func(next http.Handler) http.Handler) http.Handler {
	middlewares = excludeNils(middlewares)

	// If there's no middleware then just return the bare endpoint
	if len(middlewares) == 0 {
		return endpoint
	}

	// We wrap middleware in reverse order so that they are applied in the correct order.
	// Consider the analogy of a wrapped present. The first wrapping that you see (the outside)
	// needs to be applied last; and last wrapping that you see needs to be applied first.
	slices.Reverse(middlewares)
	for _, m := range middlewares {
		endpoint = m(endpoint)
	}

	return endpoint
}

func excludeNils(input []func(http.Handler) http.Handler) []func(http.Handler) http.Handler {
	output := make([]func(http.Handler) http.Handler, 0, len(input))
	for _, m := range input {
		if m != nil {
			output = append(output, m)
		}
	}
	return output
}
