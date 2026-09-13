package server

import (
	"errors"
	"strings"
	"testing"
)

// TestRenderPagePreservesNginxVariables locks down the regression that
// motivated this file: the replacement string used to go through
// regexp.ReplaceAllString, which treats "$name" as a capture-group reference
// and expanded every nginx variable to the empty string.
func TestRenderPagePreservesNginxVariables(t *testing.T) {
	body := strings.Join([]string{
		"proxy_set_header Host $host;",
		"proxy_set_header X-Real-IP $remote_addr;",
		"proxy_pass http://$upstream_addr;",
		"set $a ${b}c;",
		"log_format main '$status $body_bytes_sent';",
	}, "\n")

	got := renderPage(body)

	for _, want := range []string{
		"$host", "$remote_addr", "$upstream_addr", "${b}c", "$status", "$body_bytes_sent",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered page dropped %q; the replacement must use ReplaceAllLiteralString", want)
		}
	}
}

// TestRenderPageEscapesMarkup checks that caller-supplied text cannot close the
// textarea early and inject markup into the page.
func TestRenderPageEscapesMarkup(t *testing.T) {
	got := renderPage(`</textarea><script>alert(1)</script>`)

	// The document legitimately contains its own <script> tags (it loads
	// CodeMirror), so assert on the injected payload rather than on the tag.
	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Error("the injected script tag survived unescaped")
	}
	if !strings.Contains(got, "&lt;/textarea&gt;&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("rendered page did not escape the injected markup")
	}
	// Exactly one textarea element must survive.
	if n := strings.Count(got, "<textarea"); n != 1 {
		t.Errorf("expected 1 <textarea, got %d", n)
	}
	if n := strings.Count(got, "</textarea>"); n != 1 {
		t.Errorf("expected 1 </textarea>, got %d", n)
	}
}

// TestRenderPageIsPure guards the cross-visitor leak: rendering must not mutate
// shared state, so one request's config can never be served to another.
func TestRenderPageIsPure(t *testing.T) {
	pristine := PAGE_DOCUMENT

	first := renderPage("server { listen 80; }")
	if PAGE_DOCUMENT != pristine {
		t.Fatal("renderPage mutated PAGE_DOCUMENT")
	}

	second := renderPage("events { }")
	if strings.Contains(second, "listen 80") {
		t.Error("a later render leaked the earlier caller's input")
	}
	if !strings.Contains(first, "listen 80") {
		t.Error("the first render lost its own input")
	}
}

func TestFormatPage(t *testing.T) {
	t.Run("uses the formatter output", func(t *testing.T) {
		fn := func(s string, indent int, char string) (string, error) {
			return "formatted:" + s, nil
		}
		got := formatPage("  http { }  ", 2, " ", fn)
		if !strings.Contains(got, "formatted:http { }") {
			t.Error("formatPage did not render the formatter output, or did not trim its input")
		}
	})

	t.Run("reports a formatter failure in the textarea", func(t *testing.T) {
		fn := func(s string, indent int, char string) (string, error) {
			return "", errors.New("boom")
		}
		got := formatPage("junk", 2, " ", fn)
		if !strings.Contains(got, "format error") {
			t.Error("formatPage did not surface the formatting failure")
		}
	})
}
