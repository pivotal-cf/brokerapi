// Package blog is the brokerapi logger
// BrokerAPI was originally written to use the CloudFoundry Lager logger (https://github.com/cloudfoundry/lager),
// and it relied on some idiosyncrasies of that logger that are not found in the (subsequently written)
// Go standard library log/slog logger. This package is a wrapper around log/slog that adds back the
// idiosyncrasies of lager, minimizes boilerplate code, and keeps the behavior as similar as possible.
package blog

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pivotal-cf/brokerapi/v10/middlewares"
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

func (b Blog) Error(message string, err error, attr ...any) {
	b.logger.Error(join(b.prefix, message), append([]any{slog.Any(errorKey, err)}, attr...)...)
}

func (b Blog) Info(message string, attr ...any) {
	b.logger.Info(join(b.prefix, message), attr...)
}

func (b Blog) With(attr ...any) Blog {
	b.logger = b.logger.With(attr...)
	return b
}

func InstanceID(instanceID string) slog.Attr {
	return slog.String(instanceIDLogKey, instanceID)
}

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
