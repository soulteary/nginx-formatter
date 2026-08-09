package nginx

import (
	"fmt"
	"strings"
)

// Parser is a recursive-descent parser producing an nginx AST.
type Parser struct {
	lex      *Lexer
	tok      Token
	lastLine int // line of the most recently consumed (non-lookahead) token
}

// Parse lexes and parses src into a *Config. It returns an error with a line
// number on unbalanced braces or a missing semicolon.
func Parse(src string) (*Config, error) {
	p := &Parser{lex: NewLexer(src)}
	p.advance()
	nodes, err := p.parseNodes(false, 0)
	if err != nil {
		return nil, err
	}
	if p.tok.Type == TokenRBrace {
		return nil, fmt.Errorf("line %d: unexpected '}' without matching '{'", p.tok.Line)
	}
	return &Config{Nodes: nodes}, nil
}

func (p *Parser) advance() {
	p.tok = p.lex.Next()
}

// parseNodes parses a sequence of nodes until EOF (top level) or a matching
// '}' (when inBlock is true). openLine is the source line of the opening '{'
// (0 at the top level) and is used to preserve a leading blank line.
func (p *Parser) parseNodes(inBlock bool, openLine int) ([]Node, error) {
	var nodes []Node
	prevLine := openLine // line where the previous statement (or '{') ended
	first := true

	for {
		switch p.tok.Type {
		case TokenEOF:
			if inBlock {
				return nil, fmt.Errorf("line %d: unexpected EOF, missing '}'", p.tok.Line)
			}
			return nodes, nil
		case TokenRBrace:
			if inBlock {
				// Preserve a blank line just before the closing '}'.
				if !first && p.tok.Line-prevLine >= 2 {
					nodes = append(nodes, &BlankLine{})
				}
				return nodes, nil
			}
			// Handled by caller (Parse) to report a clear error.
			return nodes, nil
		}

		// Detect blank line(s) between statements and preserve as a single
		// BlankLine node (collapsing multiple blank lines into one). A leading
		// blank line (before the first statement) is preserved too, relative to
		// the top of the file or the opening '{'.
		if first {
			if p.tok.Line-openLine >= 2 {
				nodes = append(nodes, &BlankLine{})
			}
		} else if p.tok.Line-prevLine >= 2 {
			nodes = append(nodes, &BlankLine{})
		}
		first = false

		node, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		prevLine = p.lastLine
	}
}

// parseStatement parses a comment, a directive, or a block.
func (p *Parser) parseStatement() (Node, error) {
	if p.tok.Type == TokenComment {
		c := &Comment{Text: p.tok.Value}
		p.lastLine = p.tok.Line
		p.advance()
		return c, nil
	}

	name := ""
	var args []string
	startLine := p.tok.Line

	// First token is the directive/block name.
	if p.tok.Type == TokenIdent || p.tok.Type == TokenString {
		name = p.tok.Value
		p.lastLine = p.tok.Line
		p.advance()
	}

	// Collect arguments up to ';', '{', '}', comment or EOF.
	for {
		switch p.tok.Type {
		case TokenIdent, TokenString:
			args = append(args, p.tok.Value)
			p.lastLine = p.tok.Line
			p.advance()
			continue
		}
		break
	}

	switch p.tok.Type {
	case TokenLBrace:
		braceLine := p.tok.Line
		p.lastLine = p.tok.Line
		// OpenResty embedded script blocks (e.g. content_by_lua_block) hold
		// arbitrary Lua that must not be parsed as nginx syntax. The lexer has
		// already consumed the opening "{"; scan its body verbatim from the raw
		// character stream, then realign p.tok to the token after the closing
		// "}" via p.advance().
		if isRawBlockName(name) {
			raw, err := p.lex.ReadRawBlock(braceLine)
			if err != nil {
				return nil, err
			}
			p.lastLine = p.lex.line
			p.advance()
			blk := &RawBlock{Name: name, Args: args, Raw: raw}
			// A comment on the same line as the closing "}" attaches to the
			// close-brace line (rendered as `} # comment`).
			blk.InlineComment = p.consumeInlineComment()
			return blk, nil
		}
		p.advance()
		// A comment on the same line as the opening "{" attaches to the
		// open-brace line (rendered as `head { # comment`).
		openComment := ""
		if p.tok.Type == TokenComment && p.tok.Line == braceLine {
			openComment = p.tok.Value
			p.lastLine = p.tok.Line
			p.advance()
		}
		body, err := p.parseNodes(true, braceLine)
		if err != nil {
			return nil, err
		}
		if p.tok.Type != TokenRBrace {
			return nil, fmt.Errorf("line %d: missing '}' for block %q opened", p.tok.Line, name)
		}
		closeLine := p.tok.Line
		p.lastLine = p.tok.Line
		p.advance()
		blk := &Block{Name: name, Args: args, Body: body, OpenComment: openComment}
		// A comment on the same line as the closing "}" attaches to the
		// close-brace line (rendered as `} # comment`).
		if p.tok.Type == TokenComment && p.tok.Line == closeLine {
			blk.InlineComment = p.tok.Value
			p.lastLine = p.tok.Line
			p.advance()
		}
		return blk, nil

	case TokenSemicolon:
		p.lastLine = p.tok.Line
		p.advance()
		dir := &Directive{Name: name, Args: args}
		dir.InlineComment = p.consumeInlineComment()
		return dir, nil

	case TokenComment:
		// A comment appearing where a ';' or '{' is expected: treat as a
		// standalone/inline comment but the directive is missing its ';'.
		return nil, fmt.Errorf("line %d: missing ';' after directive %q", p.tok.Line, name)

	case TokenRBrace, TokenEOF:
		return nil, fmt.Errorf("line %d: missing ';' after directive %q", startLine, name)

	default:
		return nil, fmt.Errorf("line %d: unexpected token %s", p.tok.Line, p.tok.Type)
	}
}

// isRawBlockName reports whether a block directive name denotes an OpenResty
// embedded script block whose body must be preserved verbatim rather than
// parsed as nginx syntax. Detection is by suffix so all *_by_lua_block
// directives are covered without hardcoding an enumeration.
func isRawBlockName(name string) bool {
	return strings.HasSuffix(name, "_by_lua_block")
}

// consumeInlineComment attaches a trailing comment to a statement when it is
// on the same source line as the statement's terminator.
func (p *Parser) consumeInlineComment() string {
	if p.tok.Type == TokenComment && p.tok.Line == p.lastLine {
		text := p.tok.Value
		p.lastLine = p.tok.Line
		p.advance()
		return text
	}
	return ""
}
