package eventually

type EventApplier interface {
	Apply(Event) error
}
