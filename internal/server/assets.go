package server

import (
	_ "embed"
	"html"
	"regexp"
	"strings"
)

//go:embed assets/base.css
var PAGE_STYLESTHEET string
var CACHE_STYLESHEET = []byte(PAGE_STYLESTHEET)

//go:embed assets/base.js
var PAGE_SCRIPT string
var CACHE_SCRIPT = []byte(PAGE_SCRIPT)

//go:embed assets/index.html
var PAGE_DOCUMENT string

var REGEXP_UPDATE_CODE = regexp.MustCompile(`(?m)<textarea id="code" name="code">([\s\S]+)<\/textarea>`)

// renderPage returns the WebUI document with the textarea contents replaced by
// body. It is pure: the page is handed to one caller and never stored, so
// concurrent visitors cannot observe each other's input.
//
// Two details are load-bearing and should not be "simplified" back:
//
//   - ReplaceAllLiteralString, not ReplaceAllString. In a replacement string
//     "$name" is a capture-group reference, so ReplaceAllString expands $host,
//     $remote_addr and $upstream_addr to the empty string — silently deleting
//     the most common construct in an nginx config.
//   - html.EscapeString. body is caller-supplied and is spliced into an HTML
//     document; without escaping, a "</textarea>" in the input closes the
//     element early and the remainder is parsed as markup.
func renderPage(body string) string {
	return REGEXP_UPDATE_CODE.ReplaceAllLiteralString(
		PAGE_DOCUMENT,
		`<textarea id="code" name="code">`+html.EscapeString(body)+`</textarea>`,
	)
}

// formatPage formats s and renders the resulting page. A formatting failure is
// reported inside the textarea rather than as an HTTP error, matching the
// previous behaviour.
//
// The parser's message (which carries the offending line number) is included:
// a bare "format error" gives the user nothing to act on. renderPage escapes
// it, so the text is inert in the page.
func formatPage(s string, indent int, char string, fn func(s string, indent int, char string) (string, error)) string {
	formatted, err := fn(strings.TrimSpace(s), indent, char)
	if err != nil {
		formatted = "# format error: " + err.Error() + "\n\n" + strings.TrimSpace(s)
	}
	return renderPage(formatted)
}
