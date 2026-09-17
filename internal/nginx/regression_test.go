package nginx

import (
	"strings"
	"testing"
)

// format is a helper for the round-trip assertions below.
func format(t *testing.T, src string) string {
	t.Helper()
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return Format(cfg, 2, " ")
}

// TestMidWordQuotesAreLiteral covers the lexer rule that used to differ from
// nginx: a quote only opens a string when it is the first character of a
// token. Splitting a bare word at an embedded quote changed a directive's
// argument count, and nginx then refused to load the formatted file.
func TestMidWordQuotesAreLiteral(t *testing.T) {
	cases := []string{
		`sub_filter href="/old" href="/new";`,
		`add_header X-Tag a"b";`,
		`alias /data/o'brien/;`,
		`set $a x"y"z;`,
		`limit_req_zone $binary_remote_addr zone=one:10m"x";`,
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			got := strings.TrimSuffix(format(t, src), "\n")
			if got != src {
				t.Errorf("argument was split and rejoined with a space:\n got: %s\nwant: %s", got, src)
			}
		})
	}
}

// TestMidWordQuoteDoesNotSwallowFollowingDirectives is the more damaging half
// of the same bug: with two mid-word quotes the lexer ran a string token from
// the first to the second, absorbing every directive in between.
func TestMidWordQuoteDoesNotSwallowFollowingDirectives(t *testing.T) {
	src := "a o'x;\nb o'y;\n"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cfg.Nodes) != 2 {
		t.Fatalf("expected 2 directives, got %d: %#v", len(cfg.Nodes), cfg.Nodes)
	}
}

// TestEmptyBlockKeepsComments locks down the printer's compact "{  }" form: it
// may only be used when there is no comment to carry, because on re-parse a
// comment after it binds to the closing brace and a second pass would drop it.
func TestEmptyBlockKeepsComments(t *testing.T) {
	cases := []string{
		"server {\n} # close\n",
		"server { # open\n}\n",
		"server { # open\n} # close\n",
		"content_by_lua_block {\n} # close\n",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			out := format(t, src)
			if !strings.Contains(out, "#") {
				t.Fatalf("comment deleted: %q", out)
			}
			if again := format(t, out); again != out {
				t.Errorf("not idempotent:\npass1: %q\npass2: %q", out, again)
			}
		})
	}
}

// TestEmptyBlockWithoutCommentStaysCompact guards the other direction: the
// compact form is still used when nothing needs carrying.
func TestEmptyBlockWithoutCommentStaysCompact(t *testing.T) {
	if got := format(t, "events {}"); got != "events {  }\n" {
		t.Errorf("expected the compact form, got %q", got)
	}
}

// TestFormatIsIdempotent runs a corpus twice and requires a fixed point. A
// formatter that is not idempotent produces churn diffs in CI and, here, used
// to delete a second comment on the next run.
func TestFormatIsIdempotent(t *testing.T) {
	corpus := []string{
		"user nginx;\nworker_processes auto;\n",
		"http {\n  server {\n    listen 80;\n  }\n}\n",
		"events {}\n",
		"server {\n} # note\n",
		"# leading\n\n\nhttp { # open\n  include /etc/nginx/conf.d/*.conf;\n} # close\n",
		"map $host $x {\n  default 0;\n  ~^www\\. 1;\n}\n",
		"location / {\n  proxy_set_header Host $host;\n  proxy_pass http://up;\n}\n",
		"content_by_lua_block {\n  local t = {a = 1}\n  ngx.say(\"ok\")\n}\n",
		"upstream b {\n} # TODO\n",
		"server {\n  return 301 https://$host$request_uri;\n}\n",
	}
	for _, src := range corpus {
		t.Run(src, func(t *testing.T) {
			one := format(t, src)
			two := format(t, one)
			if one != two {
				t.Errorf("not a fixed point:\npass1: %q\npass2: %q", one, two)
			}
		})
	}
}

// TestFormatEndsWithSingleNewline pins the file terminator. It used to depend
// on whether the blank-line pass happened to fire on the last line, so configs
// ending in a directive lost their trailing newline entirely.
func TestFormatEndsWithSingleNewline(t *testing.T) {
	for _, src := range []string{
		"user nginx;",
		"http {\n  server {\n    listen 80;\n  }\n}",
		"events {}",
	} {
		got := format(t, src)
		if !strings.HasSuffix(got, "\n") {
			t.Errorf("%q: output does not end with a newline: %q", src, got)
		}
		if strings.HasSuffix(got, "\n\n") {
			t.Errorf("%q: output ends with more than one newline: %q", src, got)
		}
	}
}

// TestParseRejectsNULByte covers the lexer's EOF sentinel: rune 0 meant both
// "end of input" and "NUL byte", so a NUL silently truncated the config, which
// the CLI then wrote back over the original file.
func TestParseRejectsNULByte(t *testing.T) {
	if _, err := Parse("a 1;\x00b 2;"); err == nil {
		t.Fatal("expected an error for input containing a NUL byte")
	}
}

// TestParseRejectsInvalidUTF8 covers the other lossy conversion: []rune()
// replaces each invalid byte with U+FFFD, so a GBK-encoded config came back
// with its non-ASCII bytes destroyed.
func TestParseRejectsInvalidUTF8(t *testing.T) {
	if _, err := Parse("# \xd6\xd0\nserver { listen 80; }"); err == nil {
		t.Fatal("expected an error for input that is not valid UTF-8")
	}
}

// TestParseAcceptsValidUTF8 guards against over-correcting: non-ASCII comments
// are perfectly legal in an nginx config.
func TestParseAcceptsValidUTF8(t *testing.T) {
	src := "# 中文注释\nserver {\n  listen 80;\n}\n"
	if !strings.Contains(format(t, src), "中文注释") {
		t.Error("a valid UTF-8 comment was lost")
	}
}

// TestParseRejectsExcessiveNesting bounds the O(depth^2) indentation blow-up
// that let one small unauthenticated POST exhaust memory.
func TestParseRejectsExcessiveNesting(t *testing.T) {
	deep := strings.Repeat("a{", MaxBlockDepth+10) + strings.Repeat("}", MaxBlockDepth+10)
	if _, err := Parse(deep); err == nil {
		t.Fatalf("expected an error past %d levels of nesting", MaxBlockDepth)
	}

	// A realistic depth must still be accepted.
	ok := strings.Repeat("a{\n", 10) + "listen 80;\n" + strings.Repeat("}\n", 10)
	if _, err := Parse(ok); err != nil {
		t.Errorf("10 levels of nesting should be fine, got: %v", err)
	}
}

// FuzzParseFormat asserts the two properties a formatter must hold for every
// input it accepts: Format's output re-parses, and formatting is a fixed point
// after one pass. Run it with, e.g.:
//
//	go test ./internal/nginx -fuzz=FuzzParseFormat -fuzztime=60s
func FuzzParseFormat(f *testing.F) {
	for _, seed := range []string{
		"user nginx;",
		"events {}",
		"http {\n  server {\n    listen 80;\n  }\n}",
		"server {\n} # note",
		"server { # open\n} # close",
		"map $host $x {\n  default 0;\n}",
		"sub_filter href=\"/a\" href=\"/b\";",
		"content_by_lua_block {\n  local t = {a = 1}\n}",
		"# comment\n\n\nuser nginx;",
		"location ~ ^/api/(.*)$ {\n  proxy_pass http://b/$1;\n}",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, src string) {
		cfg, err := Parse(src)
		if err != nil {
			return // rejecting input is fine; mangling it is not
		}
		one := Format(cfg, 2, " ")

		cfg2, err := Parse(one)
		if err != nil {
			t.Fatalf("Format produced output that does not re-parse: %v\ninput: %q\noutput: %q", err, src, one)
		}
		if two := Format(cfg2, 2, " "); two != one {
			t.Fatalf("Format is not a fixed point:\ninput: %q\npass1: %q\npass2: %q", src, one, two)
		}
	})
}
