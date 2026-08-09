package nginx

// TokenType enumerates the kinds of lexical tokens produced by the Lexer.
type TokenType int

const (
	// TokenEOF marks the end of the input.
	TokenEOF TokenType = iota
	// TokenIdent is a bare word/argument (directive name, value, variable, etc.).
	TokenIdent
	// TokenString is a single- or double-quoted string, including its quotes.
	TokenString
	// TokenSemicolon is the ";" statement terminator.
	TokenSemicolon
	// TokenLBrace is the "{" that opens a block.
	TokenLBrace
	// TokenRBrace is the "}" that closes a block.
	TokenRBrace
	// TokenComment is a "#" line comment, including the leading "#".
	TokenComment
)

// Token is a single lexical unit with its source position.
type Token struct {
	Type  TokenType
	Value string
	Line  int
}

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "Ident"
	case TokenString:
		return "String"
	case TokenSemicolon:
		return "Semicolon"
	case TokenLBrace:
		return "LBrace"
	case TokenRBrace:
		return "RBrace"
	case TokenComment:
		return "Comment"
	default:
		return "Unknown"
	}
}
