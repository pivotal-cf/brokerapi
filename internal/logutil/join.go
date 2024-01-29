package logutil

import "strings"

func Join(s ...string) string {
	return strings.Join(s, ".")
}
