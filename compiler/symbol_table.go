package compiler

type SymbolScope string

const (
	GlobalScope SymbolScope = "GLOBAL"
	LocalScope SymbolScope = "LOCAL"
	BuiltinScope SymbolScope = "BUILTIN"
	FreeScope SymbolScope = "FREE"
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

	Outer *SymbolTable
	FreeSymbols []Symbol
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol)}
}

func NewEnclosedSymbolTable(s *SymbolTable) *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol), Outer: s}
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

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok || obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
			return obj, ok
		}
		free := s.DefineFree(obj)
		return free, true
	}
	return obj, ok
}

// this is somewhat hacky
// TODO: implement a cleaner separation of declaration (let x;) and assignment (x = 2;)
func (s *SymbolTable) ResolveNoOuter(name string) (Symbol, bool) {
	sym, ok := s.store[name]
	if sym.Scope == FunctionScope {
		return Symbol{}, false
	}
	return sym, ok
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
