package parser

import (
	"fmt"
	"strconv"

	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // == !=
	LESSGREATER // < > <= >=
	SUM         // + -
	PRODUCT     // * /
	PREFIX      // -X or !X
	FNCALL      // myFunc(X) or arr[1])
	INDEX       // hashMap["key"]
)

var tokenPriorityMap map[token.TokenType]int = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NEQ:      EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LTE:      LESSGREATER,
	token.GTE:      LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.ASTERISK: PRODUCT,
	token.SLASH:    PRODUCT,
	token.LPAREN:   FNCALL,
	token.LBRACKET: FNCALL,
}

var builtinFunctions []token.TokenType = []token.TokenType{
	token.LEN,
	token.PUTS,
	token.PUSH,
	token.POP,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	lexer     *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		lexer:          l,
		errors:         []string{},
		prefixParseFns: make(map[token.TokenType]prefixParseFn),
		infixParseFns:  make(map[token.TokenType]infixParseFn),
	}

	// read 2 tokens to set curToken and peekToken
	p.nextToken()
	p.nextToken()

	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixUnaryOp)
	p.registerPrefix(token.MINUS, p.parsePrefixUnaryOp)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	for _, tokenType := range builtinFunctions {
		p.registerPrefix(tokenType, p.parseBuiltinFunction)
	}

	for tokenType := range tokenPriorityMap {
		p.registerInfix(tokenType, p.parseInfixBinaryOp)
	}

	// override Parser.parseInfixBinaryOp for special syntax
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexAccess)

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) peekError(tokenType token.TokenType) {
	p.errors = append(p.errors, fmt.Sprintf("expected next token to be %s, got=%q", tokenType, p.peekToken.Type))
}

func (p *Parser) expectPeek(tokenType token.TokenType) bool {
	if p.peekToken.Type == tokenType {
		p.nextToken()
		return true
	}
	p.peekError(tokenType)
	return false
}

// lexer takes input, so we already have the program as input
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	var stmt ast.Statement
	switch {
	case p.curToken.Type == token.LET:
		stmt = p.parseLetStatement()
	case p.curToken.Type == token.RETURN:
		stmt = p.parseReturnStatement()
	case p.curToken.Type == token.FOR:
		stmt = p.parseForStatement()
	case p.curToken.Type == token.BREAK:
		stmt = p.parseBreakStatement()
	case p.curToken.Type == token.CONTINUE:
		stmt = p.parseContinueStatement()
	case p.curToken.Type == token.IDENT && p.peekToken.Type == token.ASSIGN:
		stmt = p.parseAssignmentStatement()
	default:
		stmt = p.parseExpressionStatement()
	}

	if p.peekToken.Type == token.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpression(priority int) ast.Expression {
	var expr ast.Expression
	if prefix, ok := p.prefixParseFns[p.curToken.Type]; ok {
		expr = prefix()
	} else {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	for p.peekToken.Type != token.SEMICOLON && priority < tokenPriorityMap[p.peekToken.Type] {
		p.nextToken()
		expr = p.infixParseFns[p.curToken.Type](expr)
	}

	return expr
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	ls := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	ls.Identifier = &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	if p.peekToken.Type == token.SEMICOLON {
		return ls
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	if expr := p.parseExpression(LOWEST); expr != nil {
		ls.Rhs = expr
		if fnLit, ok := expr.(*ast.FunctionLiteral); ok {
			fnLit.Name = ls.Identifier.Value
		}
	} else {
		return nil
	}
	return ls
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	as := &ast.AssignmentStatement{
		Identifier: &ast.Identifier{p.curToken, p.curToken.Literal},
		Token: p.curToken,
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()

	if expr := p.parseExpression(LOWEST); expr != nil {
		as.Rhs = expr
	} else {
		return nil
	}
	return as
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	rs := &ast.ReturnStatement{Token: p.curToken}

	if p.peekToken.Type == token.SEMICOLON {
		return rs
	}

	p.nextToken()

	if expr := p.parseExpression(LOWEST); expr != nil {
		rs.ReturnValue = expr
	} else {
		return nil
	}

	return rs
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	if expr := p.parseExpression(LOWEST); expr != nil {
		exprStmt := &ast.ExpressionStatement{Token: p.curToken, Expression: expr}
		return exprStmt
	}
	return nil
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // { ->

	blockStmt := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	for p.curToken.Type != token.RBRACE && p.curToken.Type != token.EOF {
		if nextStmt := p.parseStatement(); nextStmt != nil {
			blockStmt.Statements = append(blockStmt.Statements, nextStmt)
		} else {
			return nil
		}
		p.nextToken()
	}

	return blockStmt
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	return &ast.BreakStatement{Token: p.curToken}
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	return &ast.ContinueStatement{Token: p.curToken}
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	forStmt := &ast.ForStatement{Token: p.curToken}

	if p.peekToken.Type != token.LBRACE {
		p.nextToken()
		if expr := p.parseExpression(LOWEST); expr != nil {
			forStmt.Condition = expr
		} else {
			return nil
		}
	}

	if blockStmt := p.parseBlockStatement(); blockStmt != nil {
		forStmt.Body = blockStmt
	} else {
		return nil
	}

	return forStmt
}

func (p *Parser) parseIfExpression() ast.Expression {
	ifExpr := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	if expr := p.parseExpression(LOWEST); expr != nil {
		ifExpr.Condition = expr
	} else {
		return nil
	}

	if blockStmt := p.parseBlockStatement(); blockStmt != nil {
		ifExpr.Consequence = blockStmt
	} else {
		return nil
	}
	p.nextToken()
	if p.curToken.Type == token.ELSE {
		if blockStmt := p.parseBlockStatement(); blockStmt != nil {
			ifExpr.Alternative = blockStmt
		} else {
			return nil
		}
	}
	if p.peekToken.Type == token.SEMICOLON {
		p.nextToken()
	}

	return ifExpr
}

func (p *Parser) registerPrefix(t token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[t] = fn
}

func (p *Parser) registerInfix(t token.TokenType, fn infixParseFn) {
	p.infixParseFns[t] = fn
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBuiltinFunction() ast.Expression {
	return &ast.BuiltinFunction{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as int64", p.curToken.Literal))
		return nil
	}
	return &ast.IntegerLiteral{Token: p.curToken, Value: value}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	if p.curToken.Type == token.TRUE {
		return &ast.BooleanLiteral{Token: p.curToken, Value: true}
	} else if p.curToken.Type == token.FALSE {
		return &ast.BooleanLiteral{Token: p.curToken, Value: false}
	}
	p.errors = append(p.errors, fmt.Sprintf("could not parse token %+v as boolean", p.curToken))
	return nil
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{}
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	p.errors = append(p.errors, fmt.Sprintf("no prefix parse function for %s found", t))
}

func (p *Parser) parsePrefixUnaryOp() ast.Expression {
	unaryOp := &ast.PrefixUnaryOp{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	if expr := p.parseExpression(PREFIX); expr != nil {
		unaryOp.Rhs = expr
		return unaryOp
	}
	return nil
}

func (p *Parser) parseInfixBinaryOp(lhs ast.Expression) ast.Expression {
	binaryOp := &ast.InfixBinaryOp{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Lhs:      lhs,
	}

	priority, ok := tokenPriorityMap[p.curToken.Type]
	if !ok {
		p.errors = append(p.errors, fmt.Sprintf("no priority given for infix operator token %+v", p.curToken))
		return nil
	}

	p.nextToken()

	if expr := p.parseExpression(priority); expr != nil {
		binaryOp.Rhs = expr
		return binaryOp
	}
	return nil
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	if expr := p.parseExpression(LOWEST); expr != nil {
		if p.expectPeek(token.RPAREN) {
			return expr
		}
	}

	return nil
}

// curToken: FUNCTION
// peekToken: LPAREN
func (p *Parser) parseFunctionLiteral() ast.Expression {
	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	fnLit := &ast.FunctionLiteral{Token: p.curToken}
	fnLit.Parameters = p.parseFunctionParameters()
	if fnLit.Parameters == nil {
		return nil
	}
	fnLit.Body = p.parseBlockStatement()
	if fnLit.Body == nil {
		return nil
	}

	return fnLit
}

// curToken: LBRACKET
// peekToken: <Expression> | RBRACKET
func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{Token: p.curToken}

	if contents := p.parseCommaSeparatedExpressions(); contents != nil {
		arr.Contents = contents
		return arr
	}
	return nil
}

// curToken: LBRACE
// peekToken: <Expression> | RBRACE
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}

	if p.peekToken.Type == token.RBRACE {
		p.nextToken()
		return hash
	}

	p.nextToken()

	if firstHashPair := p.parseHashPair(); firstHashPair.Key != nil && firstHashPair.Value != nil {
		hash.Contents = []ast.HashPair{firstHashPair}
	} else {
		return nil
	}

	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		if hashPair := p.parseHashPair(); hashPair.Key != nil && hashPair.Value != nil {
			hash.Contents = append(hash.Contents, hashPair)
		} else {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return hash
}

// curToken: <Expression>
func (p *Parser) parseHashPair() ast.HashPair {
	hashExpr := &ast.HashPair{}

	if expr := p.parseExpression(LOWEST); expr != nil {
		hashExpr.Key = expr
	}
	if !p.expectPeek(token.COLON) {
		return *hashExpr
	}
	p.nextToken()
	if expr := p.parseExpression(LOWEST); expr != nil {
		hashExpr.Value = expr
	}
	return *hashExpr
}

// curToken: LPAREN
// peekToken: IDENT | RPAREN
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekToken.Type == token.RPAREN {
		p.nextToken()
		return identifiers
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// curToken: LPAREN
// peekToken: <Expression> | RPAREN
func (p *Parser) parseCallExpression(fn ast.Expression) ast.Expression {
	callExpr := &ast.CallExpression{Token: p.curToken, Function: fn}
	if args := p.parseCommaSeparatedExpressions(); args == nil {
		return nil
	} else {
		callExpr.Arguments = args
	}

	return callExpr
}

// curToken: LBRACKET
// peekToken: <Expression>
func (p *Parser) parseIndexAccess(container ast.Expression) ast.Expression {
	idxExpr := &ast.IndexAccess{Token: p.curToken, Container: container}
	p.nextToken() // [ ->
	if index := p.parseExpression(LOWEST); index != nil {
		idxExpr.Index = index
		p.nextToken() // ] ->
		return idxExpr
	}
	return nil
}

// curToken: LPAREN | LBRACE | LBRACKET
// peekToken: <Expression> | R<curToken>
func (p *Parser) parseCommaSeparatedExpressions() []ast.Expression {
	args := []ast.Expression{}

	var closeTokenType token.TokenType
	switch p.curToken.Type {
	case token.LPAREN:
		closeTokenType = token.RPAREN
	case token.LBRACE:
		closeTokenType = token.RBRACE
	case token.LBRACKET:
		closeTokenType = token.RBRACKET
	}

	if p.peekToken.Type == closeTokenType {
		p.nextToken()
		return args
	}

	p.nextToken()

	if firstArg := p.parseExpression(LOWEST); firstArg != nil {
		args = append(args, firstArg)
	} else {
		return nil
	}

	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		if arg := p.parseExpression(LOWEST); arg != nil {
			args = append(args, arg)
		} else {
			return nil
		}
	}

	if !p.expectPeek(closeTokenType) {
		return nil
	}

	return args
}
