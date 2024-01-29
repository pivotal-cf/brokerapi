package logutil

import "log/slog"

func Error(err error) slog.Attr {
	return slog.Any("error", err.Error())
}
