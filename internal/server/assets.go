package server

import (
	_ "embed"
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
var CACHE_DOCUMENT string

var REGEXP_UPDATE_CODE = regexp.MustCompile(`(?m)<textarea id="code" name="code">([\s\S]+)<\/textarea>`)

func getDocCache() string {
	if CACHE_DOCUMENT == "" {
		return PAGE_DOCUMENT
	}
	return CACHE_DOCUMENT
}

func updateDocCache(s string, indent int, char string, fn func(s string, indent int, char string) (string, error)) {
	input := strings.TrimSpace(s)
	formatted, err := fn(input, indent, char)
	if err != nil {
		formatted = "format error"
	}
	CACHE_DOCUMENT = REGEXP_UPDATE_CODE.ReplaceAllString(PAGE_DOCUMENT, `<textarea id="code" name="code">`+formatted+`</textarea>`)
}
