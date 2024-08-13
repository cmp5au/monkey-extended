package compiler

import (
	"fmt"

	"monkey/ast"
	"monkey/code"
	"monkey/object"
	"monkey/serializer"
)

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
	controlFlow         ControlFlow
}

type ControlFlow struct {
	breakStack [][]int
	continueStack [][]int
}

type Compiler struct {
	constants []object.Object

	symbolTable *SymbolTable
	scopes      []CompilationScope
	scopeIndex  int
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		controlFlow: ControlFlow{[][]int{}, [][]int{}},
	}

	symbolTable := NewSymbolTable()
	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	return &Compiler{
		constants:   []object.Object{},
		symbolTable: symbolTable,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		controlFlow: ControlFlow{[][]int{{}}, [][]int{{}}},
	}

	return &Compiler{
		constants:   constants,
		symbolTable: s,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, stmt := range node.Statements {
			err := c.Compile(stmt)
			if err != nil {
				return err
			}
		}
	case *ast.BlockStatement:
		for _, stmt := range node.Statements {
			err := c.Compile(stmt)
			if err != nil {
				return err
			}
		}
		if len(node.Statements) == 0 {
			c.emit(code.OpNull)
			c.emit(code.OpPop)
		}
	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)
	case *ast.LetStatement:
		symbol, ok := c.symbolTable.Resolve(node.Identifier.Value, false)
		if !ok || symbol.Scope == FunctionScope {
			symbol = c.symbolTable.Define(node.Identifier.Value)
		}
		if node.Rhs == nil {
			c.emit(code.OpNull)
		} else if err := c.Compile(node.Rhs); err != nil {
			return err
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}
		c.emit(code.OpNull)
		c.emit(code.OpPop)
	case *ast.AssignmentStatement:
		symbol, ok := c.symbolTable.Resolve(node.Identifier.Value, true)
		if !ok {
			return fmt.Errorf("variable %s not declared in scope", node.Identifier.Value)
		}
		if err := c.Compile(node.Rhs); err != nil {
			return err
		}
		switch symbol.Scope {
		case GlobalScope:
			c.emit(code.OpSetGlobal, symbol.Index)
		case LocalScope:
			c.emit(code.OpSetLocal, symbol.Index)
		case FreeScope:
			c.emit(code.OpSetFree, symbol.Index)
		case BuiltinScope:
			return fmt.Errorf("cannot assign to builtin function")
		case FunctionScope:
			return fmt.Errorf("variable %s not declared in scope", symbol.Name)
		default:
			return fmt.Errorf("unknown scope: %s", symbol.Scope)
		}
		c.emit(code.OpNull)
		c.emit(code.OpPop)
	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value, true)
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value)
		}
		c.loadSymbol(symbol)
	case *ast.IfExpression:
		if err := c.Compile(node.Condition); err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)
		if err := c.Compile(node.Consequence); err != nil {
			return err
		}
		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}
		if !c.lastInstructionIsPush() {
			c.emit(code.OpNull)
		}

		jumpPos := c.emit(code.OpJump, 9999)
		c.changeOperand(jumpNotTruthyPos, len(c.scopes[c.scopeIndex].instructions))

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}
			if !c.lastInstructionIsPush() {
				c.emit(code.OpNull)
			}
		}
		c.changeOperand(jumpPos, len(c.scopes[c.scopeIndex].instructions))
	case *ast.ForStatement:
		cf := &c.scopes[c.scopeIndex].controlFlow
		cf.breakStack = append(cf.breakStack, []int{})
		cf.continueStack = append(cf.continueStack, []int{})
		loopStart := len(c.scopes[c.scopeIndex].instructions)
		jumpNotTruthyPos := -1
		if node.Condition != nil {
			if err := c.Compile(node.Condition); err != nil {
				return err
			}
			
			jumpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)
		}
		if err := c.Compile(node.Body); err != nil {
			return err
		}
		c.emit(code.OpJump, loopStart)
		if jumpNotTruthyPos != -1 {
			c.changeOperand(jumpNotTruthyPos, len(c.scopes[c.scopeIndex].instructions))
		}
		for _, breakPos := range cf.breakStack[len(cf.breakStack) - 1] {
			c.changeOperand(breakPos, len(c.scopes[c.scopeIndex].instructions))
		}
		for _, continuePos := range cf.continueStack[len(cf.continueStack) - 1] {
			c.changeOperand(continuePos, loopStart)
		}
		cf.breakStack = cf.breakStack[:len(cf.breakStack)-1]
		cf.continueStack = cf.continueStack[:len(cf.continueStack)-1]
		c.emit(code.OpNull)
		c.emit(code.OpPop)
	case *ast.BreakStatement:
		cf := &c.scopes[c.scopeIndex].controlFlow
		if len(cf.breakStack) == 0 {
			return fmt.Errorf("cannot break without an enclosing `for` loop")
		}
		cf.breakStack[len(cf.breakStack)-1] = append(
			cf.breakStack[len(cf.breakStack)-1],
			c.emit(code.OpJump, 9999),
		)
	case *ast.ContinueStatement:
		cf := &c.scopes[c.scopeIndex].controlFlow
		if len(cf.continueStack) == 0 {
			return fmt.Errorf("cannot continue without an enclosing `for` loop")
		}
		cf.continueStack[len(cf.continueStack)-1] = append(
			cf.continueStack[len(cf.continueStack)-1],
			c.emit(code.OpJump, 9999),
		)
	case *ast.PrefixUnaryOp:
		err := c.Compile(node.Rhs)
		if err != nil {
			return err
		}
		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown unary operator %s", node.Operator)
		}
	case *ast.InfixBinaryOp:
		if node.Operator == ">" || node.Operator == ">=" {
			err := c.Compile(node.Rhs)
			if err != nil {
				return err
			}
			err = c.Compile(node.Lhs)
			if err != nil {
				return err
			}
		} else {
			err := c.Compile(node.Lhs)
			if err != nil {
				return err
			}
			err = c.Compile(node.Rhs)
			if err != nil {
				return err
			}
		}
		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "==":
			c.emit(code.OpEq)
		case "!=":
			c.emit(code.OpNeq)
		case "<", ">":
			c.emit(code.OpLessThan)
		case "<=", ">=":
			c.emit(code.OpLessThanEq)
		default:
			return fmt.Errorf("unknown binary operator %s", node.Operator)
		}
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.BooleanLiteral:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	case *ast.StringLiteral:
		c.emit(code.OpConstant, c.addConstant(&object.String{Value: node.Value}))
	case *ast.NullLiteral:
		c.emit(code.OpNull)
	case *ast.ArrayLiteral:
		for _, content := range node.Contents {
			c.Compile(content)
		}
		c.emit(code.OpArray, len(node.Contents))
	case *ast.HashLiteral:
		for _, contentPair := range node.Contents {
			c.Compile(contentPair.Key)
			c.Compile(contentPair.Value)
		}
		c.emit(code.OpHash, len(node.Contents))
	case *ast.IndexAccess:
		c.Compile(node.Container)
		c.Compile(node.Index)
		c.emit(code.OpIndex)
	case *ast.FunctionLiteral:
		c.enterScope()

		if node.Name != "" {
			c.symbolTable.DefineFunctionName(node.Name)
		}

		for _, param := range node.Parameters {
			c.symbolTable.Define(param.Value)
		}

		if err := c.Compile(node.Body); err != nil {
			return err
		}
		if c.scopes[c.scopeIndex].lastInstruction.Opcode == code.OpPop {
			lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
			c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))
			c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
		}
		if c.scopes[c.scopeIndex].lastInstruction.Opcode != code.OpReturnValue {
			c.emit(code.OpReturn)
		}
		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()

		for _, sym := range freeSymbols {
			c.loadSymbol(sym)
		}
		compiledFn := &object.CompiledFunction{
			Instructions: instructions,
			NumLocals:    numLocals,
			NumParameters: len(node.Parameters),
		}
		c.emit(code.OpClosure, c.addConstant(compiledFn), len(freeSymbols))
	case *ast.ReturnStatement:
		if err := c.Compile(node.ReturnValue); err != nil {
			return err
		}
		c.emit(code.OpReturnValue)
	case *ast.CallExpression:
		// builtin calls handled separately because they can be
		// variadic, see push for an example
		if builtin, ok := node.Function.(*ast.BuiltinFunction); ok {
			return c.compileBuiltinCall(builtin, node.Arguments)
		}
			
		if err := c.Compile(node.Function); err != nil {
			return err
		}

		for _, a := range node.Arguments {
			if err := c.Compile(a); err != nil {
				return err
			}
		}
		c.emit(code.OpCall, len(node.Arguments))
	default:
		return fmt.Errorf("unknown node type %T (%+v)", node, node)
	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.scopes[c.scopeIndex].instructions,
		Constants:    c.constants,
	}
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.scopes[c.scopeIndex].instructions)
	c.scopes[c.scopeIndex].instructions = append(c.scopes[c.scopeIndex].instructions, ins...)
	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIsPop() bool {
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == code.OpPop
}

func (c *Compiler) lastInstructionIsPush() bool {
	nonPushers := []code.Opcode{
		code.OpPop,
		code.OpJump,
		code.OpJumpNotTruthy,
		code.OpSetGlobal,
		code.OpSetLocal,
	}

	for _, np := range nonPushers {
		if c.scopes[c.scopeIndex].lastInstruction.Opcode == np {
			return false
		}
	}
	return true
}

func (c *Compiler) removeLastPop() {
	curScope := &c.scopes[c.scopeIndex]
	curScope.instructions = curScope.instructions[:curScope.lastInstruction.Position]
	curScope.lastInstruction = curScope.previousInstruction
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.scopes[c.scopeIndex].instructions[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.scopes[c.scopeIndex].instructions[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		controlFlow:         ControlFlow{[][]int{}, [][]int{}},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.scopes[c.scopeIndex].instructions

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	case FunctionScope:
		c.emit(code.OpCurrentClosure)
	}
}

func (c *Compiler) compileBuiltinCall(
	builtin *ast.BuiltinFunction,
	args []ast.Expression,
) error {
	builtinSymbol, ok := c.symbolTable.Resolve(builtin.Value, true)
	if !ok {
		return fmt.Errorf("unable to resolve builtin %s", builtin.Value)
	}
	c.loadSymbol(builtinSymbol)

	// 1. put all args on the stack
	// 2. squash them into a single array arg
	// 3. call builtin function with a single array argument
	for _, arg := range args {
		if err := c.Compile(arg); err != nil {
			return err
		}
	}
	c.emit(code.OpArray, len(args))
	c.emit(code.OpCall, 1)
	return nil
}

func (b *Bytecode) Serialize() []byte {
	buf := []byte{}
	for _, c := range b.Constants {
		switch c := c.(type) {
		case *object.Integer:
			buf = append(buf, c.Serialize()...)
		case *object.String:
			buf = append(buf, c.Serialize()...)
		case *object.CompiledFunction:
			buf = append(buf, c.Serialize()...)
		default:
			panic(fmt.Sprintf("cannot serialize constant of type %T", c))
		}
	}
	// main instructions don't come with length since they are the last chunk
	buf = append(buf, byte(serializer.BYTECODE))
	buf = append(buf, b.Instructions...)
	return buf
}

func (b *Bytecode) Deserialize(bs []byte) int {
	constants := []object.Object{}
	i := 0
	previ := -1
	for i < len(bs) {
		if i == previ {
			panic(fmt.Sprintf("made no progress at index %d: %x", i, bs[i]))
		}
		previ = i
		switch bs[i] {
		case byte(serializer.INTEGER):
			x := &object.Integer{}
			n := x.Deserialize(bs[i:])
			if n < 9 {
				panic(fmt.Sprintf("bad integer deserialization, got %d bytes: %v", n, bs[i:i+9]))
			}
			i += n
			constants = append(constants, x)
		case byte(serializer.STRING):
			s := &object.String{}
			n := s.Deserialize(bs[i:])
			if n < 9 {
				panic(fmt.Sprintf("bad string deserialization, got %d bytes: %v", n, bs[i:i+9]))
			}
			i += n
			constants = append(constants, s)
		case byte(serializer.COMPILEDFN):
			f := &object.CompiledFunction{}
			n := f.Deserialize(bs[i:])
			if n < 25 {
				panic(fmt.Sprintf("bad compiled function deserialization, got %d bytes: %v", n, bs[i:]))
			}
			i += n
			constants = append(constants, f)
		case byte(serializer.BYTECODE):
			b.Constants = constants
			b.Instructions = bs[i + 1:]
			return len(bs)
		default:
			panic(fmt.Sprintf("can't deserialize ObjectSerialType %d", bs[i]))
		}
	}
	panic("reached the end of the bytecode without decoding instructions")
}
