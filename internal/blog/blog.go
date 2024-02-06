// Package blog is the brokerapi logger
// BrokerAPI was originally written to use the CloudFoundry Lager logger, and it relied on some idiosyncrasies
// of that logger that are not supported by the (subsequently written) standard library log/slog logger.
// This package is a wrapper around log/slog that adds back the idiosyncrasies of lager, so that the behavior
// is exactly the same.
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

func New(ctx context.Context, logger *slog.Logger, prefix string, supplementary ...slog.Attr) Blog {
	var attr []any
	for _, s := range supplementary {
		attr = append(attr, s)
	}

	for _, key := range []middlewares.ContextKey{middlewares.CorrelationIDKey, middlewares.RequestIdentityKey} {
		if value := ctx.Value(key); value != nil {
			attr = append(attr, slog.Any(string(key), value))
		}
	}

	return Blog{
		logger: logger.With(attr...),
		prefix: prefix,
	}
}

func (b Blog) Error(message string, err error) {
	b.logger.Error(join(b.prefix, message), slog.Any(errorKey, err))
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
