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

func (e *Environment) Get(name string, local bool) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && !local && e.parent != nil {
		obj, ok = e.parent.Get(name, local)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object, setIfAbsent bool) Object {
	_, ok := e.Get(name, true)
	if !setIfAbsent && !ok && e.parent != nil {
		return e.parent.Set(name, val, setIfAbsent)
	}

	e.store[name] = val
	return val
}
