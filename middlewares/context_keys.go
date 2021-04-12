package middlewares

type ContextKey string

const (
	CorrelationIDKey       ContextKey = "correlation-id"
	InfoLocationKey        ContextKey = "info-location"
	OriginatingIdentityKey ContextKey = "originating-id"
	RequestIdentityKey     ContextKey = "request-id"
)
