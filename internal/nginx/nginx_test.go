package nginx

import (
	"reflect"
	"strings"
	"testing"
)

func tokenTypes(src string) []Token {
	l := NewLexer(src)
	var toks []Token
	for {
		t := l.Next()
		toks = append(toks, t)
		if t.Type == TokenEOF {
			return toks
		}
	}
}

func TestLexerBasic(t *testing.T) {
	toks := tokenTypes("listen 80;")
	want := []TokenType{TokenIdent, TokenIdent, TokenSemicolon, TokenEOF}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i].Type != w {
			t.Errorf("token %d: got %s, want %s", i, toks[i].Type, w)
		}
	}
	if toks[0].Value != "listen" || toks[1].Value != "80" {
		t.Errorf("unexpected values: %v", toks)
	}
}

func TestLexerHashInsideString(t *testing.T) {
	// The '#' inside the double-quoted string must NOT start a comment.
	toks := tokenTypes(`add_header X-Color "#ffffff";`)
	if len(toks) != 5 {
		t.Fatalf("got %d tokens, want 5: %v", len(toks), toks)
	}
	if toks[2].Type != TokenString || toks[2].Value != `"#ffffff"` {
		t.Errorf("string token wrong: %+v", toks[2])
	}
	if toks[3].Type != TokenSemicolon {
		t.Errorf("expected semicolon, got %s", toks[3].Type)
	}
}

func TestLexerComment(t *testing.T) {
	toks := tokenTypes("# a comment\nlisten 80;")
	if toks[0].Type != TokenComment || toks[0].Value != " a comment" {
		t.Errorf("comment token wrong: %+v", toks[0])
	}
}

func TestLexerSingleAndDoubleQuotes(t *testing.T) {
	toks := tokenTypes(`log_format m '$remote_addr "$request"';`)
	if toks[2].Type != TokenString || toks[2].Value != `'$remote_addr "$request"'` {
		t.Errorf("single-quoted string wrong: %+v", toks[2])
	}
}

func TestLexerVariablesPreserved(t *testing.T) {
	toks := tokenTypes("return 200 ${scheme}://$host;")
	if toks[1].Value != "200" {
		t.Errorf("arg wrong: %+v", toks[1])
	}
	if toks[2].Type != TokenIdent || toks[2].Value != "${scheme}://$host" {
		t.Errorf("variable token not preserved verbatim: %+v", toks[2])
	}
	if toks[3].Type != TokenSemicolon {
		t.Errorf("expected semicolon, got %+v", toks[3])
	}
}

func TestLexerSubFilterWithSpaces(t *testing.T) {
	src := `sub_filter '<a href="http://foo">' '<a href="https://bar">';`
	toks := tokenTypes(src)
	if toks[0].Value != "sub_filter" {
		t.Errorf("directive name wrong: %+v", toks[0])
	}
	if toks[1].Type != TokenString || toks[1].Value != `'<a href="http://foo">'` {
		t.Errorf("first arg wrong: %+v", toks[1])
	}
	if toks[2].Type != TokenString || toks[2].Value != `'<a href="https://bar">'` {
		t.Errorf("second arg wrong: %+v", toks[2])
	}
}

func TestLexerEscapeInString(t *testing.T) {
	toks := tokenTypes(`return 200 "a\"b";`)
	if toks[2].Type != TokenString || toks[2].Value != `"a\"b"` {
		t.Errorf("escaped quote not handled: %+v", toks[2])
	}
}

func TestLexerEscapedSpaceInBareWord(t *testing.T) {
	// A backslash-escaped space inside a bare word (e.g. an nginx regex) must
	// not split the word into two idents. The escape is kept verbatim.
	toks := tokenTypes(`(Edg|Sogou\ web|Curl)`)
	if toks[0].Type != TokenIdent || toks[0].Value != `(Edg|Sogou\ web|Curl)` {
		t.Errorf("escaped space in bare word not preserved: %+v", toks[0])
	}
	if toks[1].Type != TokenEOF {
		t.Errorf("expected single ident then EOF, got %+v", toks[1])
	}
}

func TestFormatEscapedSpaceInRegexArg(t *testing.T) {
	src := "server {\n" +
		"if ($http_user_agent ~ (Edg|Sogou\\ web|Semrushbot|Scrapy|Curl)) { return 403; }\n" +
		"}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	first := Format(cfg, 4, " ")
	if !strings.Contains(first, `(Edg|Sogou\ web|Semrushbot|Scrapy|Curl)`) {
		t.Errorf("escaped space in regex arg not kept intact:\n%s", first)
	}

	cfg2, err := Parse(first)
	if err != nil {
		t.Fatalf("re-parse error: %v\noutput was:\n%s", err, first)
	}
	second := Format(cfg2, 4, " ")
	if first != second {
		t.Errorf("format not idempotent.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestParseNestedBlocks(t *testing.T) {
	cfg, err := Parse("http {\nserver {\nlisten 80;\n}\n}")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(cfg.Nodes) != 1 {
		t.Fatalf("expected 1 top node, got %d", len(cfg.Nodes))
	}
	http, ok := cfg.Nodes[0].(*Block)
	if !ok || http.Name != "http" {
		t.Fatalf("expected http block, got %#v", cfg.Nodes[0])
	}
	server, ok := http.Body[0].(*Block)
	if !ok || server.Name != "server" {
		t.Fatalf("expected server block, got %#v", http.Body[0])
	}
	dir, ok := server.Body[0].(*Directive)
	if !ok || dir.Name != "listen" || !reflect.DeepEqual(dir.Args, []string{"80"}) {
		t.Fatalf("expected listen 80, got %#v", server.Body[0])
	}
}

func TestParseEmptyBlock(t *testing.T) {
	cfg, err := Parse("events {}")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	blk, ok := cfg.Nodes[0].(*Block)
	if !ok || blk.Name != "events" || len(blk.Body) != 0 {
		t.Fatalf("expected empty events block, got %#v", cfg.Nodes[0])
	}
}

func TestParseCommentPositions(t *testing.T) {
	cfg, err := Parse("# top\nlisten 80; # inline\n# after")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if c, ok := cfg.Nodes[0].(*Comment); !ok || c.Text != " top" {
		t.Fatalf("expected top comment, got %#v", cfg.Nodes[0])
	}
	dir, ok := cfg.Nodes[1].(*Directive)
	if !ok || dir.InlineComment != " inline" {
		t.Fatalf("expected inline comment on directive, got %#v", cfg.Nodes[1])
	}
	if c, ok := cfg.Nodes[2].(*Comment); !ok || c.Text != " after" {
		t.Fatalf("expected trailing comment, got %#v", cfg.Nodes[2])
	}
}

func TestParseUnbalancedBraces(t *testing.T) {
	if _, err := Parse("http {\nlisten 80;"); err == nil {
		t.Error("expected error for missing '}'")
	}
	if _, err := Parse("listen 80;\n}"); err == nil {
		t.Error("expected error for extra '}'")
	}
}

func TestParseMissingSemicolon(t *testing.T) {
	if _, err := Parse("http {\nlisten 80\n}"); err == nil {
		t.Error("expected error for missing ';'")
	}
}

func TestRobustnessBacktickAndBraceVar(t *testing.T) {
	// The old goja approach broke on backticks (template literal delimiters)
	// and ${...}. The AST pipeline must round-trip them verbatim.
	src := "location / {\nreturn 200 \"${scheme}://`host`\";\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Format(cfg, 4, " ")
	want := "location / {\n    return 200 \"${scheme}://`host`\";\n}\n"
	if out != want {
		t.Errorf("robustness format mismatch.\n got: %q\nwant: %q", out, want)
	}
}

func TestFormatEmptyBlockFold(t *testing.T) {
	cfg, err := Parse("events {}")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Format(cfg, 4, " ")
	if out != "events {  }" {
		t.Errorf("empty block not folded: %q", out)
	}
}

func TestFormatTabIndent(t *testing.T) {
	cfg, err := Parse("http {\nlisten 80;\n}")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Format(cfg, 1, "\t")
	want := "http {\n\tlisten 80;\n}\n"
	if out != want {
		t.Errorf("tab indent mismatch.\n got: %q\nwant: %q", out, want)
	}
}

func TestParseLuaBlockBasic(t *testing.T) {
	cfg, err := Parse(`content_by_lua_block { ngx.say("hi") }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(cfg.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(cfg.Nodes))
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if rb.Name != "content_by_lua_block" {
		t.Errorf("name wrong: %q", rb.Name)
	}
	if rb.Raw != ` ngx.say("hi") ` {
		t.Errorf("raw not preserved verbatim: %q", rb.Raw)
	}
}

func TestParseLuaBlockNestedBraces(t *testing.T) {
	src := "rewrite_by_lua_block { if x then y() end; t = {a=1} }"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if rb.Raw != " if x then y() end; t = {a=1} " {
		t.Errorf("nested braces not preserved: %q", rb.Raw)
	}
}

func TestParseLuaBlockBraceInString(t *testing.T) {
	// A "}" inside a Lua string must not close the block early.
	src := `content_by_lua_block { ngx.say("}") }`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if rb.Raw != ` ngx.say("}") ` {
		t.Errorf("brace inside string closed block early: %q", rb.Raw)
	}
}

func TestParseLuaBlockUnterminated(t *testing.T) {
	if _, err := Parse("content_by_lua_block { ngx.say('hi')"); err == nil {
		t.Error("expected error for unterminated raw block")
	}
}

func TestFormatLuaBlockIdempotent(t *testing.T) {
	src := "server {\n" +
		"    location /lua {\n" +
		"        content_by_lua_block {\n" +
		"            local t = {a=1}\n" +
		"            ngx.say(\"}\")\n" +
		"        }\n" +
		"    }\n" +
		"}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	first := Format(cfg, 4, " ")

	cfg2, err := Parse(first)
	if err != nil {
		t.Fatalf("re-parse error: %v\noutput was:\n%s", err, first)
	}
	second := Format(cfg2, 4, " ")
	if first != second {
		t.Errorf("format not idempotent.\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// The embedded Lua should be reindented under level+1 and preserved.
	if !strings.Contains(first, "            local t = {a=1}") {
		t.Errorf("lua body not reindented as expected:\n%s", first)
	}
	if !strings.Contains(first, "ngx.say(\"}\")") {
		t.Errorf("lua body content missing:\n%s", first)
	}
}

func TestFormatLuaBlockInline(t *testing.T) {
	cfg, err := Parse(`content_by_lua_block { ngx.say("hi") }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Format(cfg, 4, " ")
	want := "content_by_lua_block {\n    ngx.say(\"hi\")\n}\n"
	if out != want {
		t.Errorf("inline lua block format mismatch.\n got: %q\nwant: %q", out, want)
	}
}

func TestParseNonLuaBlockRegression(t *testing.T) {
	// A normal block whose name does not end in _by_lua_block must still be
	// parsed as nginx syntax, not as a raw block.
	cfg, err := Parse("location / {\nreturn 200;\n}")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	blk, ok := cfg.Nodes[0].(*Block)
	if !ok || blk.Name != "location" {
		t.Fatalf("expected normal *Block, got %#v", cfg.Nodes[0])
	}
	if _, ok := blk.Body[0].(*Directive); !ok {
		t.Fatalf("expected directive body, got %#v", blk.Body[0])
	}
}

// --- S1: comments on the close-brace line vs open-brace line ---

func TestBlockCloseBraceComment(t *testing.T) {
	src := "http {\n    listen 80;\n} # after close"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	blk, ok := cfg.Nodes[0].(*Block)
	if !ok {
		t.Fatalf("expected *Block, got %#v", cfg.Nodes[0])
	}
	if blk.InlineComment != " after close" {
		t.Errorf("close-brace comment not captured on InlineComment: %q", blk.InlineComment)
	}
	if blk.OpenComment != "" {
		t.Errorf("unexpected open-brace comment: %q", blk.OpenComment)
	}
	out := Format(cfg, 4, " ")
	want := "http {\n    listen 80;\n} # after close\n"
	if out != want {
		t.Errorf("close-brace comment rendered wrong.\n got: %q\nwant: %q", out, want)
	}
}

func TestBlockOpenBraceComment(t *testing.T) {
	src := "http { # on open\n    listen 80;\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	blk, ok := cfg.Nodes[0].(*Block)
	if !ok {
		t.Fatalf("expected *Block, got %#v", cfg.Nodes[0])
	}
	if blk.OpenComment != " on open" {
		t.Errorf("open-brace comment not captured: %q", blk.OpenComment)
	}
	if blk.InlineComment != "" {
		t.Errorf("unexpected close-brace comment: %q", blk.InlineComment)
	}
	out := Format(cfg, 4, " ")
	want := "http { # on open\n    listen 80;\n}\n"
	if out != want {
		t.Errorf("open-brace comment rendered wrong.\n got: %q\nwant: %q", out, want)
	}
}

func TestBlockBothBraceComments(t *testing.T) {
	src := "http { # on open\n    listen 80;\n} # after close"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Format(cfg, 4, " ")
	want := "http { # on open\n    listen 80;\n} # after close\n"
	if out != want {
		t.Errorf("both brace comments rendered wrong.\n got: %q\nwant: %q", out, want)
	}
	// Idempotence.
	cfg2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}
	if second := Format(cfg2, 4, " "); second != out {
		t.Errorf("not idempotent.\nfirst: %q\nsecond: %q", out, second)
	}
}

func TestRawBlockCloseBraceComment(t *testing.T) {
	src := "content_by_lua_block {\n    ngx.say(\"hi\")\n} # note"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if rb.InlineComment != " note" {
		t.Errorf("close-brace comment not on InlineComment: %q", rb.InlineComment)
	}
	if strings.Contains(rb.Raw, "note") {
		t.Errorf("close-brace comment polluted Raw body: %q", rb.Raw)
	}
	first := Format(cfg, 4, " ")
	want := "content_by_lua_block {\n    ngx.say(\"hi\")\n} # note\n"
	if first != want {
		t.Errorf("raw block close comment rendered wrong.\n got: %q\nwant: %q", first, want)
	}
	// format(format(x)) == format(x): re-parse must not eat "# note" as Lua.
	cfg2, err := Parse(first)
	if err != nil {
		t.Fatalf("re-parse error: %v\noutput was:\n%s", err, first)
	}
	rb2, ok := cfg2.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock on re-parse, got %#v", cfg2.Nodes[0])
	}
	if strings.Contains(rb2.Raw, "note") {
		t.Errorf("re-parse polluted Raw body with close comment: %q", rb2.Raw)
	}
	if second := Format(cfg2, 4, " "); second != first {
		t.Errorf("raw block with close comment not idempotent.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// --- M2: Lua long strings / long comments inside raw blocks ---

func TestRawBlockLuaLongString(t *testing.T) {
	src := "content_by_lua_block {\n    local s = [[ text with } and { ]]\n    ngx.say(s)\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if !strings.Contains(rb.Raw, "[[ text with } and { ]]") {
		t.Errorf("long string not preserved verbatim: %q", rb.Raw)
	}
	first := Format(cfg, 4, " ")
	cfg2, err := Parse(first)
	if err != nil {
		t.Fatalf("re-parse error: %v\noutput:\n%s", err, first)
	}
	if second := Format(cfg2, 4, " "); second != first {
		t.Errorf("long-string raw block not idempotent.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRawBlockLuaLongStringLevel(t *testing.T) {
	src := "content_by_lua_block {\n    local s = [=[ has ]] and } inside ]=]\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if !strings.Contains(rb.Raw, "[=[ has ]] and } inside ]=]") {
		t.Errorf("leveled long string not preserved: %q", rb.Raw)
	}
}

func TestRawBlockLuaLongComment(t *testing.T) {
	src := "content_by_lua_block {\n    --[[ a } comment { block ]]\n    ngx.say(\"ok\")\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if !strings.Contains(rb.Raw, "--[[ a } comment { block ]]") {
		t.Errorf("long comment not preserved: %q", rb.Raw)
	}
}

func TestRawBlockLuaLineComment(t *testing.T) {
	// An unbalanced brace inside a "-- ..." line comment must not close early.
	src := "content_by_lua_block {\n    local x = 1 -- close } here\n    ngx.say(x)\n}"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	rb, ok := cfg.Nodes[0].(*RawBlock)
	if !ok {
		t.Fatalf("expected *RawBlock, got %#v", cfg.Nodes[0])
	}
	if !strings.Contains(rb.Raw, "-- close } here") {
		t.Errorf("line comment not preserved: %q", rb.Raw)
	}
}

// --- M3: nested ${...} variables ---

func TestLexerNestedBraceVar(t *testing.T) {
	toks := tokenTypes("set $x ${a${b}};")
	if toks[2].Type != TokenIdent || toks[2].Value != "${a${b}}" {
		t.Errorf("nested brace var not preserved: %+v", toks[2])
	}
	if toks[3].Type != TokenSemicolon {
		t.Errorf("expected semicolon after nested var, got %+v", toks[3])
	}
}

func TestParseNestedBraceVar(t *testing.T) {
	cfg, err := Parse("set $x ${a${b}};")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	dir, ok := cfg.Nodes[0].(*Directive)
	if !ok || dir.Name != "set" {
		t.Fatalf("expected set directive, got %#v", cfg.Nodes[0])
	}
	if !reflect.DeepEqual(dir.Args, []string{"$x", "${a${b}}"}) {
		t.Errorf("nested var args wrong: %#v", dir.Args)
	}
	out := Format(cfg, 4, " ")
	if out != "set $x ${a${b}};" {
		t.Errorf("nested var render wrong: %q", out)
	}
}

func TestLexerSingleBraceVarUnchanged(t *testing.T) {
	toks := tokenTypes("return 200 ${ scheme }://$host;")
	if toks[2].Type != TokenIdent || toks[2].Value != "${ scheme }://$host" {
		t.Errorf("single-level ${ } var behavior changed: %+v", toks[2])
	}
}
