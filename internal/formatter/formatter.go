package formatter

import (
	"github.com/soulteary/nginx-formatter/internal/nginx"
)

// Formatter parses the given nginx configuration and returns it formatted with
// indent copies of char per nesting level. The signature is preserved so it
// can continue to be passed as a func value by callers.
func Formatter(s string, indent int, char string) (string, error) {
	if s == "" {
		return "", nil
	}
	cfg, err := nginx.Parse(s)
	if err != nil {
		return "", err
	}
	return nginx.Format(cfg, indent, char), nil
}
