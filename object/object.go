package object

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/cmp5au/monkey-extended/ast"
	"github.com/cmp5au/monkey-extended/code"
)

const (
	STRING            = "STRING"
	INTEGER           = "INTEGER"
	BOOLEAN           = "BOOLEAN"
	NULL              = "NULL"
	ARRAY             = "ARRAY"
	HASH              = "HASH"
	RETURN            = "RETURN"
	ERROR             = "ERROR"
	BREAK             = "BREAK"
	CONTINUE          = "CONTINUE"
	FUNCTION          = "FUNCTION"
	BUILTIN           = "BUILTIN"
	COMPILED_FUNCTION = "COMPILED_FUNCTION"
	CLOSURE           = "CLOSURE"
)

type ObjectType string

type Object interface {
	Type() ObjectType
	Inspect() string
}

type HashKey struct {
	Type    ObjectType
	KeyRepr string
	Value   uint64
}

type Hashable interface {
	Hash() HashKey
}

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING }

func (s *String) Inspect() string {
	return "\"" + s.Value + "\""
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER }

func (i *Integer) Inspect() string {
	return fmt.Sprintf("%d", i.Value)
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN }

func (b *Boolean) Inspect() string {
	return fmt.Sprintf("%t", b.Value)
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL }

func (n *Null) Inspect() string { return "null" }

type ReturnValue struct {
	Value Object
}

func (r *ReturnValue) Type() ObjectType { return RETURN }

func (r *ReturnValue) Inspect() string { return r.Value.Inspect() }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR }

func (e *Error) Inspect() string { return "ERROR: " + e.Message }

type Break struct{}

func (b *Break) Type() ObjectType { return BREAK }

func (b *Break) Inspect() string { return "break" }

type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE }

func (c *Continue) Inspect() string { return "continue" }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION }

func (f *Function) Inspect() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("fn(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n")

	return out.String()
}

type JitInstructions struct {
	sync.Mutex
	MachineCodeInstructions []byte
}

type CompiledFunction struct {
	Instructions  code.Instructions
	NumLocals     int
	NumParameters int
	*JitInstructions
}

func (c *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION }

func (c *CompiledFunction) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", c)
}

type Builtin func([]Object) Object

func (b Builtin) Type() ObjectType { return BUILTIN }

func (b Builtin) Inspect() string { return BUILTIN }

func ExposeBuiltin(bf *ast.BuiltinFunction) Object {
	for _, builtin := range Builtins {
		if bf.TokenLiteral() == builtin.Name {
			return builtin.Builtin
		}
	}
	return &Error{bf.TokenLiteral() + " is not a builtin function"}
}

type Array []Object

func (a *Array) Type() ObjectType { return ARRAY }

func (a *Array) Inspect() string {
	var out bytes.Buffer
	objStrings := []string{}

	for _, obj := range *a {
		objStrings = append(objStrings, obj.Inspect())
	}

	out.WriteString("[ ")
	out.WriteString(strings.Join(objStrings, ", "))
	out.WriteString(" ]")

	return out.String()
}

type Hash map[HashKey]Object

func (h *Hash) Type() ObjectType { return HASH }

func (h *Hash) Inspect() string {
	var out bytes.Buffer
	hashPairStrings := []string{}

	for key, val := range *h {
		hashPairStrings = append(hashPairStrings, key.KeyRepr+": "+val.Inspect())
	}
	out.WriteString("{ ")
	out.WriteString(strings.Join(hashPairStrings, ", "))
	out.WriteString(" }")

	return out.String()
}

type Closure struct {
	Fn   *CompiledFunction
	Free []Object
}

func (c *Closure) Type() ObjectType { return CLOSURE }

func (c *Closure) Inspect() string {
	return fmt.Sprintf("Closure[%p]", c)
}
