package aggregate

type Type struct {
	name    string
	factory func() Root
}

func (t Type) Name() string { return t.name }

func NewType(name string, factory func() Root) Type {
	return Type{name: name, factory: factory}
}

func (t Type) instance() Root { return t.factory() }
