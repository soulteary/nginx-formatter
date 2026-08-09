package nginx

import "fmt"

// Lexer performs character-by-character lexical analysis of an nginx config.
//
// It correctly handles single/double quoted strings (whose inner { } ; #
// characters are not treated as syntax), backslash escapes inside strings,
// "#" line comments (a "#" inside a string does not start a comment), and
// variables such as $var and ${var}, which are preserved verbatim as part of
// the surrounding ident token.
type Lexer struct {
	input []rune
	pos   int
	line  int
}

// NewLexer creates a Lexer over the given source string.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	c := l.input[l.pos]
	l.pos++
	if c == '\n' {
		l.line++
	}
	return c
}

func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == '\v'
}

// isDelimiter reports whether c terminates a bare ident token.
func isDelimiter(c rune) bool {
	return c == 0 || isSpace(c) || c == ';' || c == '{' || c == '}' || c == '#' || c == '"' || c == '\''
}

// Next returns the next token from the input.
func (l *Lexer) Next() Token {
	// Skip whitespace between tokens.
	for isSpace(l.peek()) {
		l.next()
	}

	startLine := l.line
	c := l.peek()

	switch c {
	case 0:
		return Token{Type: TokenEOF, Line: startLine}
	case ';':
		l.next()
		return Token{Type: TokenSemicolon, Value: ";", Line: startLine}
	case '{':
		l.next()
		return Token{Type: TokenLBrace, Value: "{", Line: startLine}
	case '}':
		l.next()
		return Token{Type: TokenRBrace, Value: "}", Line: startLine}
	case '#':
		return l.lexComment(startLine)
	case '"', '\'':
		return l.lexString(startLine)
	default:
		return l.lexIdent(startLine)
	}
}

// ReadRawBlock scans the verbatim body of a raw block (e.g. an OpenResty
// *_by_lua_block) starting from the position immediately after the opening
// "{" has already been consumed. It scans character by character until the
// matching "}" is found, tracking nested "{"/"}" depth.
//
// So that braces inside Lua source do not affect the depth count, it skips
// over: single/double quoted strings (with backslash escapes), Lua long
// strings/long comments delimited by "[[ ... ]]" or "[=*[ ... ]=*]" (the
// number of "=" must match to close), long comments "--[[ ... ]]", and plain
// "-- ..." line comments (skipped to end of line).
//
// On success it returns the text between the outermost braces (excluding the
// braces themselves) with the closing "}" consumed. If the block is never
// closed it returns an error carrying the line where the block opened.
func (l *Lexer) ReadRawBlock(openLine int) (string, error) {
	var sb []rune
	depth := 1
	for {
		c := l.peek()
		if c == 0 {
			return "", fmt.Errorf("line %d: unterminated raw block, missing '}'", openLine)
		}
		if c == '"' || c == '\'' {
			quote := l.next()
			sb = append(sb, quote)
			for {
				d := l.peek()
				if d == 0 {
					return "", fmt.Errorf("line %d: unterminated raw block, missing '}'", openLine)
				}
				if d == '\\' {
					sb = append(sb, l.next()) // backslash
					if l.peek() != 0 {
						sb = append(sb, l.next()) // escaped char, kept verbatim
					}
					continue
				}
				sb = append(sb, l.next())
				if d == quote {
					break
				}
			}
			continue
		}
		// Lua comments: "--[[ ... ]]" long comment or "-- ..." line comment.
		if c == '-' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '-' {
			sb = append(sb, l.next()) // first '-'
			sb = append(sb, l.next()) // second '-'
			if level, ok := l.longBracketLevel(); ok {
				if err := l.skipLongBracket(&sb, level, openLine); err != nil {
					return "", err
				}
				continue
			}
			// Plain line comment: skip to end of line (keep the newline).
			for l.peek() != 0 && l.peek() != '\n' {
				sb = append(sb, l.next())
			}
			continue
		}
		// Lua long string: "[[ ... ]]" or "[=*[ ... ]=*]".
		if c == '[' {
			if level, ok := l.longBracketLevel(); ok {
				if err := l.skipLongBracket(&sb, level, openLine); err != nil {
					return "", err
				}
				continue
			}
			sb = append(sb, l.next())
			continue
		}
		if c == '{' {
			depth++
			sb = append(sb, l.next())
			continue
		}
		if c == '}' {
			depth--
			if depth == 0 {
				l.next() // consume the matching closing '}'
				return string(sb), nil
			}
			sb = append(sb, l.next())
			continue
		}
		sb = append(sb, l.next())
	}
}

// longBracketLevel reports whether the lexer is positioned at the start of a
// Lua long-bracket opener ("[", "[=[", "[==[", ...). If so it returns the
// level (number of "=" signs) without consuming any input; otherwise it
// returns ok == false.
func (l *Lexer) longBracketLevel() (int, bool) {
	if l.peek() != '[' {
		return 0, false
	}
	i := l.pos + 1
	level := 0
	for i < len(l.input) && l.input[i] == '=' {
		level++
		i++
	}
	if i < len(l.input) && l.input[i] == '[' {
		return level, true
	}
	return 0, false
}

// skipLongBracket consumes a Lua long bracket (opener "[=*[", body, closer
// "]=*]" with a matching level) verbatim into sb. The lexer must be positioned
// at the opening "[". If the closer is never found it returns an error keyed to
// openLine.
func (l *Lexer) skipLongBracket(sb *[]rune, level, openLine int) error {
	*sb = append(*sb, l.next()) // opening '['
	for i := 0; i < level; i++ {
		*sb = append(*sb, l.next()) // '='
	}
	*sb = append(*sb, l.next()) // second '['
	for {
		c := l.peek()
		if c == 0 {
			return fmt.Errorf("line %d: unterminated raw block, missing '}'", openLine)
		}
		if c == ']' {
			// Look ahead for a matching "]=*]" of the same level.
			i := l.pos + 1
			cnt := 0
			for i < len(l.input) && l.input[i] == '=' {
				cnt++
				i++
			}
			if cnt == level && i < len(l.input) && l.input[i] == ']' {
				*sb = append(*sb, l.next()) // closing ']'
				for j := 0; j < level; j++ {
					*sb = append(*sb, l.next()) // '='
				}
				*sb = append(*sb, l.next()) // final ']'
				return nil
			}
		}
		*sb = append(*sb, l.next())
	}
}

func (l *Lexer) lexComment(startLine int) Token {
	l.next() // consume leading '#'
	var sb []rune
	for {
		c := l.peek()
		if c == 0 || c == '\n' {
			break
		}
		sb = append(sb, l.next())
	}
	return Token{Type: TokenComment, Value: string(sb), Line: startLine}
}

func (l *Lexer) lexString(startLine int) Token {
	quote := l.next() // consume opening quote
	var sb []rune
	sb = append(sb, quote)
	for {
		c := l.peek()
		if c == 0 {
			// Unterminated string: return what we have; parser may still use it.
			break
		}
		if c == '\\' {
			sb = append(sb, l.next()) // backslash
			if l.peek() != 0 {
				sb = append(sb, l.next()) // escaped char, kept verbatim
			}
			continue
		}
		if c == quote {
			sb = append(sb, l.next()) // closing quote
			break
		}
		sb = append(sb, l.next())
	}
	return Token{Type: TokenString, Value: string(sb), Line: startLine}
}

// lexIdent reads a bare word. A "${...}" variable reference is kept intact
// (its inner braces are not treated as syntax), and quoted strings embedded
// directly against the word are not split, so arguments like zone=foo:10m and
// $var/${var} are preserved verbatim.
func (l *Lexer) lexIdent(startLine int) Token {
	var sb []rune
	for {
		c := l.peek()
		// Handle "${...}" so the inner braces aren't taken as block syntax.
		// Nested braces such as "${a${b}}" are supported by tracking depth and
		// only stopping at the matching outermost "}".
		if c == '$' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '{' {
			sb = append(sb, l.next()) // '$'
			sb = append(sb, l.next()) // '{'
			depth := 1
			for l.peek() != 0 && depth > 0 {
				d := l.peek()
				if d == '{' {
					depth++
				} else if d == '}' {
					depth--
					if depth == 0 {
						sb = append(sb, l.next()) // matching '}'
						break
					}
				}
				sb = append(sb, l.next())
			}
			continue
		}
		// A backslash escapes the next character (e.g. "\ " a literal space in a
		// regex), keeping it part of the current bare word rather than a token
		// boundary. Kept verbatim so the escape is preserved on output.
		if c == '\\' {
			sb = append(sb, l.next()) // backslash
			if l.peek() != 0 {
				sb = append(sb, l.next()) // escaped char, kept verbatim
			}
			continue
		}
		if isDelimiter(c) {
			break
		}
		sb = append(sb, l.next())
	}
	return Token{Type: TokenIdent, Value: string(sb), Line: startLine}
}
