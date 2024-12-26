package postgres

import "github.com/get-eventually/go-eventually/aggregate"

// Option can be used to change the configuration of an object.
type Option[T any] interface {
	apply(T)
}

type option[T any] func(T)

func newOption[T any](f func(T)) option[T] { return option[T](f) }

func (apply option[T]) apply(val T) { apply(val) }

const (
	// DefaultAggregateTableName is the default Aggregate table name an AggregateRepository points to.
	DefaultAggregateTableName = "aggregates"
	// DefaultEventsTableName is the default Domain Events table name an AggregateRepository points to.
	DefaultEventsTableName = "events"
	// DefaultStreamsTableName is the default Event Streams table name an AggregateRepository points to.
	DefaultStreamsTableName = "event_streams"
)

// WithAggregateTableName allows you to specify a different Aggregate table name
// that an AggregateRepository should manage.
func WithAggregateTableName[ID aggregate.ID, T aggregate.Root[ID]](
	tableName string,
) Option[*AggregateRepository[ID, T]] {
	return newOption(func(repository *AggregateRepository[ID, T]) {
		repository.aggregateTableName = tableName
	})
}

// WithEventsTableName allows you to specify a different Events table name
// that an AggregateRepository should manage.
func WithEventsTableName[ID aggregate.ID, T aggregate.Root[ID]](tableName string) Option[*AggregateRepository[ID, T]] {
	return newOption(func(repository *AggregateRepository[ID, T]) {
		repository.eventsTableName = tableName
	})
}

// WithStreamsTableName allows you to specify a different Event Streams table name
// that an AggregateRepository should manage.
func WithStreamsTableName[ID aggregate.ID, T aggregate.Root[ID]](tableName string) Option[*AggregateRepository[ID, T]] {
	return newOption(func(repository *AggregateRepository[ID, T]) {
		repository.streamsTableName = tableName
	})
}
