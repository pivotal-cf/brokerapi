package middlewares

type ContextKey string

const (
	CorrelationIDKey       ContextKey = "correlation-id"
	InfoLocationKey        ContextKey = "infoLocation"
	OriginatingIdentityKey ContextKey = "originatingIdentity"
	RequestIdentityKey     ContextKey = "requestIdentity"
)
