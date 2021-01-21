package aggregate

// Type represents the type of an Aggregate, which will expose the
// name of the Aggregate (used as Event Store type) and a factory method
// to create new instances of the type, without using reflection.
type Type struct {
	name    string
	factory func() Root
}

// Name is the name of the Aggregate.
func (t Type) Name() string { return t.name }

// NewType creates a new Aggregate type.
//
// Consider creating a global variable in the package containing the Aggregate,
// and make sure the name used for the Aggregate is unique in your system,
// as to avoid clashed with other Aggregate types.
func NewType(name string, factory func() Root) Type {
	return Type{name: name, factory: factory}
}

func (t Type) instance() Root { return t.factory() }
