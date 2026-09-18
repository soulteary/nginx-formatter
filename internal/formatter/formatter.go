package formatter

import (
	"strings"

	"github.com/soulteary/nginx-formatter/internal/nginx"
)

// Formatter parses the given nginx configuration and returns it formatted with
// indent copies of char per nesting level. The signature is preserved so it
// can continue to be passed as a func value by callers.
//
// CRLF input is normalized to LF for parsing and converted back afterwards, so
// a file keeps the line endings it arrived with. Without this the result came
// back *mixed*: the printer joins structural lines with "\n", but a "\r" inside
// a comment, a raw block body or a multi-line quoted string is part of that
// token's text and survived into the output.
func Formatter(s string, indent int, char string) (string, error) {
	if s == "" {
		return "", nil
	}

	crlf := strings.Contains(s, "\r\n")
	if crlf {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}

	cfg, err := nginx.Parse(s)
	if err != nil {
		return "", err
	}

	out := nginx.Format(cfg, indent, char)
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return out, nil
}
