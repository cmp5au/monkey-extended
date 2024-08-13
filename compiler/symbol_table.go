package compiler

type SymbolScope string

const (
	GlobalScope   SymbolScope = "GLOBAL"
	LocalScope    SymbolScope = "LOCAL"
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	store          map[string]Symbol
	numDefinitions int

	Outer       *SymbolTable
	FreeSymbols []Symbol
	depth       int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol)}
}

func NewEnclosedSymbolTable(s *SymbolTable) *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol), Outer: s, depth: s.depth + 1}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}
	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) Resolve(name string, resolveIfNonlocal bool) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil && resolveIfNonlocal {
		obj, ok = s.Outer.Resolve(name, true)
		if ok && (obj.Scope == LocalScope || obj.Scope == FreeScope) {
			free := s.DefineFree(obj)
			return free, true
		}
	}
	return obj, ok
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFree(original Symbol) Symbol {
	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols), Scope: FreeScope}
	s.store[original.Name] = symbol

	s.FreeSymbols = append(s.FreeSymbols, original)
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	fnSymbol := Symbol{Name: name, Scope: FunctionScope, Index: 0}
	s.store[name] = fnSymbol
	return fnSymbol
}
