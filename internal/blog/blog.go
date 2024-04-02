// Package blog is the brokerapi logger
// BrokerAPI was originally written to use the CloudFoundry Lager logger (https://github.com/cloudfoundry/lager),
// and it relied on some idiosyncrasies of that logger that are not found in the (subsequently written)
// Go standard library log/slog logger. This package is a wrapper around log/slog that adds back the
// idiosyncrasies of lager, minimizes boilerplate code, and keeps the behavior as similar as possible.
// It also implements the slog.Handler interface so that it can easily be converted into a slog.Logger.
// This is useful when calling public APIs (such as FailureResponse.ValidatedStatusCode) which take a
// slog.Logger as an input, and because they are public cannot take a Blog as an input.
package blog

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

const (
	instanceIDLogKey = "instance-id"
	bindingIDLogKey  = "binding-id"
	errorKey         = "error"
)

type Blog struct {
	logger *slog.Logger
	prefix string
}

func New(logger *slog.Logger) Blog {
	return Blog{logger: logger}
}

// Session emulates a Lager logger session. It returns a new logger that will always log the
// attributes, prefix, and data from the context.
func (b Blog) Session(ctx context.Context, prefix string, attr ...any) Blog {
	for _, key := range []middlewares.ContextKey{middlewares.CorrelationIDKey, middlewares.RequestIdentityKey} {
		if value := ctx.Value(key); value != nil {
			attr = append(attr, slog.Any(string(key), value))
		}
	}

	return Blog{
		logger: b.logger.With(attr...),
		prefix: appendPrefix(b.prefix, prefix),
	}
}

// Error logs an error. It takes an error type as a convenience, which is different to slog.Logger.Error()
func (b Blog) Error(message string, err error, attr ...any) {
	b.logger.Error(join(b.prefix, message), append([]any{slog.Any(errorKey, err)}, attr...)...)
}

// Info logs information. It behaves a lot file slog.Logger.Info()
func (b Blog) Info(message string, attr ...any) {
	b.logger.Info(join(b.prefix, message), attr...)
}

// With returns a logger that always logs the specified attributes
func (b Blog) With(attr ...any) Blog {
	b.logger = b.logger.With(attr...)
	return b
}

// Enabled is required implement the slog.Handler interface
func (b Blog) Enabled(context.Context, slog.Level) bool {
	return true
}

// WithAttrs is required implement the slog.Handler interface
func (b Blog) WithAttrs(attrs []slog.Attr) slog.Handler {
	var attributes []any
	for _, a := range attrs {
		attributes = append(attributes, a)
	}
	return b.With(attributes...)
}

// WithGroup is required implement the slog.Handler interface
func (b Blog) WithGroup(string) slog.Handler {
	return b
}

// Handle is required implement the slog.Handler interface
func (b Blog) Handle(_ context.Context, record slog.Record) error {
	msg := join(b.prefix, record.Message)
	switch record.Level {
	case slog.LevelDebug:
		b.logger.Debug(msg)
	case slog.LevelInfo:
		b.logger.Info(msg)
	case slog.LevelWarn:
		b.logger.Warn(msg)
	default:
		b.logger.Error(msg)
	}

	return nil
}

// InstanceID creates an attribute from an instance ID
func InstanceID(instanceID string) slog.Attr {
	return slog.String(instanceIDLogKey, instanceID)
}

// BindingID creates an attribute from an binding ID
func BindingID(bindingID string) slog.Attr {
	return slog.String(bindingIDLogKey, bindingID)
}

func join(s ...string) string {
	return strings.Join(s, ".")
}

func appendPrefix(existing, addition string) string {
	switch existing {
	case "":
		return addition
	default:
		return join(existing, addition)
	}
}
