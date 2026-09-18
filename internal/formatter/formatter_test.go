package formatter_test

import (
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
)

func TestFormatter(t *testing.T) {

	const TestData = `
load_module modules/ngx_http_js_module.so;

events {  }

http {
js_path "/etc/nginx/njs/";

js_import main from http/api/set_keyval.js;

keyval_zone zone=foo:10m;

server {
listen 80;

location /keyval {
js_content main.set_keyval;
}
location /api {
internal;
api write=on;
}
location /api/ro {
api;
}
}
}`

	const TestExpected = `
load_module modules/ngx_http_js_module.so;

events {  }

http {
    js_path "/etc/nginx/njs/";

    js_import main from http/api/set_keyval.js;

    keyval_zone zone=foo:10m;

    server {
        listen 80;

        location /keyval {
            js_content main.set_keyval;
        }

        location /api {
            internal;
            api write=on;
        }

        location /api/ro {
            api;
        }

    }
}`

	result, err := formatter.Formatter(TestData, 4, " ")
	if err != nil {
		t.Errorf("formatter error: %v\n", err)
	}

	// Format terminates every file with exactly one newline.
	if result != TestExpected+"\n" {
		t.Error("formatter result not expected", result, TestExpected)
	}
}

// TestFormatterPreservesLineEndings covers the mixed-line-ending bug: the
// printer joins structural lines with "\n", but a "\r" inside a comment, a raw
// block body or a multi-line quoted string is part of that token's text, so a
// CRLF file came back with both kinds of ending in it.
func TestFormatterPreservesLineEndings(t *testing.T) {
	t.Run("CRLF stays CRLF throughout", func(t *testing.T) {
		in := "server {\r\n  # a comment\r\n  listen 80;\r\n}\r\n"
		out, err := formatter.Formatter(in, 2, " ")
		if err != nil {
			t.Fatalf("Formatter: %v", err)
		}
		if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
			t.Errorf("output mixes line endings: %q", out)
		}
		if !strings.Contains(out, "# a comment\r\n") {
			t.Errorf("comment did not keep its CRLF: %q", out)
		}
	})

	t.Run("CRLF inside a raw block", func(t *testing.T) {
		in := "content_by_lua_block {\r\n  ngx.say(\"x\")\r\n}\r\n"
		out, err := formatter.Formatter(in, 2, " ")
		if err != nil {
			t.Fatalf("Formatter: %v", err)
		}
		if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
			t.Errorf("output mixes line endings: %q", out)
		}
	})

	t.Run("LF stays LF", func(t *testing.T) {
		out, err := formatter.Formatter("server {\n  listen 80;\n}\n", 2, " ")
		if err != nil {
			t.Fatalf("Formatter: %v", err)
		}
		if strings.Contains(out, "\r") {
			t.Errorf("a CR appeared in an LF file: %q", out)
		}
	})
}
