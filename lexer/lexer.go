package lexer

import (
	"github.com/cmp5au/monkey-extended/token"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char at position
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readInt() string {
	position := l.position
	for isNumber(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	position := l.position
	l.readChar()
	for l.ch != '"' {
		l.readChar()
	}
	l.readChar()
	return l.input[position+1 : l.position-1]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	switch {
	case l.ch == '=':
		l.readChar()
		if l.ch == '=' {
			l.readChar()
			return token.Token{token.EQ, "=="}
		} else {
			return token.Token{token.ASSIGN, "="}
		}
	case l.ch == '!':
		l.readChar()
		if l.ch == '=' {
			l.readChar()
			return token.Token{token.NEQ, "!="}
		} else {
			return token.Token{token.BANG, "!"}
		}
	case l.ch == '<':
		l.readChar()
		if l.ch == '=' {
			l.readChar()
			return token.Token{token.LTE, "<="}
		} else {
			return token.Token{token.LT, "<"}
		}
	case l.ch == '>':
		l.readChar()
		if l.ch == '=' {
			l.readChar()
			return token.Token{token.GTE, ">="}
		} else {
			return token.Token{token.GT, ">"}
		}
	case l.ch == '+':
		tok = token.Token{token.PLUS, string(l.ch)}
	case l.ch == '-':
		tok = token.Token{token.MINUS, string(l.ch)}
	case l.ch == '/':
		tok = token.Token{token.SLASH, string(l.ch)}
	case l.ch == '*':
		tok = token.Token{token.ASTERISK, string(l.ch)}
	case l.ch == ',':
		tok = token.Token{token.COMMA, string(l.ch)}
	case l.ch == ':':
		tok = token.Token{token.COLON, string(l.ch)}
	case l.ch == ';':
		tok = token.Token{token.SEMICOLON, string(l.ch)}
	case l.ch == '(':
		tok = token.Token{token.LPAREN, string(l.ch)}
	case l.ch == ')':
		tok = token.Token{token.RPAREN, string(l.ch)}
	case l.ch == '{':
		tok = token.Token{token.LBRACE, string(l.ch)}
	case l.ch == '}':
		tok = token.Token{token.RBRACE, string(l.ch)}
	case l.ch == '[':
		tok = token.Token{token.LBRACKET, string(l.ch)}
	case l.ch == ']':
		tok = token.Token{token.RBRACKET, string(l.ch)}
	case l.ch == 0:
		tok = token.Token{token.EOF, ""}
	case isLetter(l.ch):
		ident := l.readIdentifier()
		return token.Token{token.LookupIdent(ident), ident}
	case isNumber(l.ch):
		return token.Token{token.INT, l.readInt()}
	case l.ch == '"':
		return token.Token{token.STRING, l.readString()}
	default:
		tok = token.Token{token.ILLEGAL, string(l.ch)}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\n' || l.ch == '\r' || l.ch == '\t' {
		l.readChar()
	}
}

func isLetter(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || b == '_'
}

func isNumber(b byte) bool {
	return '0' <= b && b <= '9'
}
