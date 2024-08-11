package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// identifiers and literals
	IDENT  = "IDENT"
	INT    = "INT"
	STRING = "\""

	// operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	SLASH    = "/"
	ASTERISK = "*"
	BANG     = "!"

	// comparators
	EQ  = "=="
	NEQ = "!="
	LT  = "<"
	GT  = ">"
	LTE = "<="
	GTE = ">="

	// delimiters
	COMMA     = ","
	COLON     = ":"
	SEMICOLON = ";"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"

	// keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	RETURN   = "RETURN"
	IF       = "IF"
	ELSE     = "ELSE"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	FOR      = "FOR"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"

	// builtin functions
	LEN  = "LEN"
	PUTS = "PUTS"
	PUSH = "PUSH"
)

var keywords = map[string]TokenType{
	"fn":       FUNCTION,
	"let":      LET,
	"return":   RETURN,
	"if":       IF,
	"else":     ELSE,
	"true":     TRUE,
	"false":    FALSE,
	"for":      FOR,
	"break":    BREAK,
	"continue": CONTINUE,
	"len":      LEN,
	"puts":     PUTS,
	"push":     PUSH,
}

func LookupIdent(ident string) TokenType {
	if keywordTok, ok := keywords[ident]; ok {
		return keywordTok
	}
	return IDENT
}
