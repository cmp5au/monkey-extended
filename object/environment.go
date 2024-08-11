package object

type Environment struct {
	store  map[string]Object
	parent *Environment
}

func NewEnvironment(parent ...*Environment) *Environment {
	if len(parent) > 1 {
		panic("Cannot pass more than a single parent to NewEnvironment")
	}
	var p *Environment
	if len(parent) == 1 {
		p = parent[0]
	}
	s := make(map[string]Object)
	return &Environment{store: s, parent: p}
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.parent != nil {
		obj, ok = e.parent.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object, setIfAbsent bool) Object {
	if setIfAbsent {
		e.store[name] = val
		return val
	}

	_, ok := e.Get(name)
	if !ok && e.parent != nil {
		return e.parent.Set(name, val, setIfAbsent)
	}
	return val
}
