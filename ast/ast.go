package ast

import (
	"bytes"
	"fmt"
	"strings"

	"monkey/token"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, stmt := range p.Statements {
		out.WriteString(stmt.String())
	}
	return out.String()
}

type LetStatement struct {
	*Identifier
	Token token.Token
	Rhs   Expression
}

func (l *LetStatement) TokenLiteral() string {
	// fmt.Printf("let statement = %+v\n", l)
	return l.Token.Literal
}

func (l *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(l.TokenLiteral() + " ")
	out.WriteString(l.Identifier.String())
	out.WriteString(" = ")

	if l.Rhs != nil {
		out.WriteString(l.Rhs.String())
	}
	out.WriteString(";")

	return out.String()
}

func (l *LetStatement) statementNode() {}

type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

func (i *Identifier) String() string { return i.Value }

func (i *Identifier) expressionNode() {}

type ReturnStatement struct {
	Token       token.Token
	ReturnValue Expression
}

func (r *ReturnStatement) TokenLiteral() string { return r.Token.Literal }

func (r *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(r.TokenLiteral() + " ")

	if r.ReturnValue != nil {
		out.WriteString(r.ReturnValue.String())
	}
	out.WriteString(";")

	return out.String()
}

func (r *ReturnStatement) statementNode() {}

type ExpressionStatement struct {
	Token token.Token
	Expression
}

func (e *ExpressionStatement) TokenLiteral() string { return e.Token.Literal }

func (e *ExpressionStatement) String() string {
	if e.Expression != nil {
		return e.Expression.String()
	}
	return ""
}

func (e *ExpressionStatement) statementNode() {}

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (i *IntegerLiteral) TokenLiteral() string { return i.Token.Literal }

func (i *IntegerLiteral) String() string { return i.Token.Literal }

func (i *IntegerLiteral) expressionNode() {}

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }

func (b *BooleanLiteral) String() string { return b.Token.Literal }

func (b *BooleanLiteral) expressionNode() {}

type StringLiteral struct {
	Token token.Token
	Value string
}

func (s *StringLiteral) TokenLiteral() string { return s.Token.Literal }

func (s *StringLiteral) String() string { return s.Token.Literal }

func (s *StringLiteral) expressionNode() {}

type PrefixUnaryOp struct {
	Token    token.Token
	Operator string
	Rhs      Expression
}

func (p *PrefixUnaryOp) TokenLiteral() string { return p.Token.Literal }

func (p *PrefixUnaryOp) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(p.Operator)
	out.WriteString(p.Rhs.String())
	out.WriteString(")")

	return out.String()
}

func (p *PrefixUnaryOp) expressionNode() {}

type InfixBinaryOp struct {
	Token    token.Token
	Operator string
	Lhs      Expression
	Rhs      Expression
}

func (i *InfixBinaryOp) TokenLiteral() string { return i.Token.Literal }

func (i *InfixBinaryOp) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(i.Lhs.String())
	out.WriteString(" " + i.Operator + " ")
	out.WriteString(i.Rhs.String())
	out.WriteString(")")

	return out.String()
}

func (i *InfixBinaryOp) expressionNode() {}

type BlockStatement struct {
	Token      token.Token
	Statements []Statement
}

func (b *BlockStatement) TokenLiteral() string { return b.Token.Literal }

func (b *BlockStatement) String() string {
	var out bytes.Buffer

	for _, stmt := range b.Statements {
		out.WriteString(stmt.String())
	}

	return out.String()
}

func (b *BlockStatement) statementNode() {}

type ForStatement struct {
	Token     token.Token
	Condition Expression
	Body      *BlockStatement
}

func (f *ForStatement) TokenLiteral() string { return f.Token.Literal }

func (f *ForStatement) String() string {
	var out bytes.Buffer

	out.WriteString("for")
	out.WriteString(f.Condition.String())
	out.WriteString(" ")
	out.WriteString(f.Body.String())

	return out.String()
}

func (f *ForStatement) statementNode() {}

type IfExpression struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (i *IfExpression) TokenLiteral() string { return i.Token.Literal }

func (i *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(i.Condition.String())
	out.WriteString(" ")
	out.WriteString(i.Consequence.String())

	if i.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(i.Alternative.String())
	}

	return out.String()
}

func (i *IfExpression) expressionNode() {}

type CallExpression struct {
	Token     token.Token
	Function  Expression
	Arguments []Expression
}

func (c *CallExpression) TokenLiteral() string { return c.Token.Literal }

func (c *CallExpression) String() string {
	var out bytes.Buffer

	out.WriteString(c.Function.String())
	out.WriteString("(")
	argStrings := make([]string, 0)
	for _, arg := range c.Arguments {
		argStrings = append(argStrings, arg.String())
	}
	out.WriteString(strings.Join(argStrings, ", "))
	out.WriteString(")")

	return out.String()
}

func (c *CallExpression) expressionNode() {}

type FunctionLiteral struct {
	Token      token.Token
	Parameters []*Identifier
	Body       *BlockStatement
	Name       string
}

func (f *FunctionLiteral) TokenLiteral() string { return f.Token.Literal }

func (f *FunctionLiteral) String() string {
	var out bytes.Buffer

	paramStrings := make([]string, 0)
	for _, param := range f.Parameters {
		paramStrings = append(paramStrings, param.String())
	}
	out.WriteString(f.TokenLiteral())
	if f.Name != "" {
		out.WriteString(fmt.Sprintf("<%s>", f.Name))
	}
	out.WriteString("(")
	out.WriteString(strings.Join(paramStrings, ", "))
	out.WriteString(") ")
	out.WriteString(f.Body.String())

	return out.String()
}

func (f *FunctionLiteral) expressionNode() {}

type BreakStatement struct {
	Token token.Token
}

func (b *BreakStatement) TokenLiteral() string { return b.Token.Literal }

func (b *BreakStatement) String() string { return b.Token.Literal }

func (b *BreakStatement) statementNode() {}

type ContinueStatement struct {
	Token token.Token
}

func (b *ContinueStatement) TokenLiteral() string { return b.Token.Literal }

func (b *ContinueStatement) String() string { return b.Token.Literal }

func (b *ContinueStatement) statementNode() {}

type BuiltinFunction struct {
	Token token.Token
	Value string
}

func (b *BuiltinFunction) TokenLiteral() string { return b.Token.Literal }

func (b *BuiltinFunction) String() string { return b.Value }

func (b *BuiltinFunction) expressionNode() {}

type ArrayLiteral struct {
	Token    token.Token
	Contents []Expression
}

func (a *ArrayLiteral) TokenLiteral() string { return a.Token.Literal }

func (a *ArrayLiteral) String() string {
	var out bytes.Buffer

	itemStrings := make([]string, 0)
	for _, item := range a.Contents {
		itemStrings = append(itemStrings, item.String())
	}
	out.WriteString("[ ")
	out.WriteString(strings.Join(itemStrings, ", "))
	out.WriteString(" ]")

	return out.String()
}

func (a *ArrayLiteral) expressionNode() {}

type HashLiteral struct {
	Token    token.Token
	Contents []HashPair
}

func (h *HashLiteral) TokenLiteral() string { return h.Token.Literal }

func (h *HashLiteral) String() string {
	var out bytes.Buffer

	itemStrings := make([]string, 0)
	for _, item := range h.Contents {
		itemStrings = append(itemStrings, item.String())
	}
	out.WriteString("[ ")
	out.WriteString(strings.Join(itemStrings, ", "))
	out.WriteString(" ]")

	return out.String()
}

func (h *HashLiteral) expressionNode() {}

type HashPair struct {
	Key   Expression
	Value Expression
}

func (h HashPair) String() string {
	return h.Key.String() + ": " + h.Value.String()
}

type IndexAccess struct {
	Token     token.Token
	Container Expression
	Index     Expression
}

func (i *IndexAccess) TokenLiteral() string { return i.Token.Literal }

func (i *IndexAccess) String() string {
	var out bytes.Buffer

	out.WriteString(i.Container.String())
	out.WriteString("[")
	out.WriteString(i.Index.String())
	out.WriteString("]")

	return out.String()
}

func (i *IndexAccess) expressionNode() {}
