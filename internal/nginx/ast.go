package nginx

// Node is any element of the nginx configuration AST.
type Node interface {
	node()
}

// Directive is a simple statement terminated by ";", such as
// "listen 80;" or "js_path \"/etc/nginx/njs/\";".
//
// InlineComment holds a trailing comment on the same line as the directive
// (without the leading "#"), if any.
type Directive struct {
	Name          string
	Args          []string
	InlineComment string
}

func (*Directive) node() {}

// Block is a directive followed by a "{ ... }" body, such as
// "http { ... }", "server { ... }" or "location /api { ... }".
//
// OpenComment holds a trailing comment on the same line as the opening "{"
// (rendered as `head { # comment`), if any. InlineComment holds a trailing
// comment on the same line as the closing "}" (rendered as `} # comment`),
// if any. Both exclude the leading "#".
type Block struct {
	Name          string
	Args          []string
	Body          []Node
	OpenComment   string
	InlineComment string
}

func (*Block) node() {}

// RawBlock is a block whose body is an opaque script (e.g. OpenResty
// *_by_lua_block). Raw holds the verbatim text between the outermost
// "{" and "}" (excluding the braces), so the embedded Lua is preserved
// rather than parsed as nginx syntax.
//
// OpenComment holds a trailing comment on the same line as the opening
// "{" (rendered as `head { # comment`), if any. InlineComment holds a
// trailing comment on the same line as the closing "}" (rendered as
// `} # comment`), if any. Both exclude the leading "#".
type RawBlock struct {
	Name          string
	Args          []string
	Raw           string
	OpenComment   string
	InlineComment string
}

func (*RawBlock) node() {}

// Comment is a standalone "#" comment line (Text excludes the leading "#").
type Comment struct {
	Text string
}

func (*Comment) node() {}

// BlankLine represents one or more blank lines in the source, used by the
// printer to preserve author-intended spacing between directives.
type BlankLine struct{}

func (*BlankLine) node() {}

// Config is the root of the AST.
type Config struct {
	Nodes []Node
}
