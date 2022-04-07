package postgresql

import "github.com/get-eventually/go-eventually/core/aggregate"

type AggregateSerializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Serializer[ID, T, []byte]
}

type AggregateDeserializer[ID aggregate.ID, T aggregate.Root[ID]] interface {
	aggregate.Deserializer[ID, T, []byte]
}

type AggregateSerde[ID aggregate.ID, T aggregate.Root[ID]] interface {
	AggregateSerializer[ID, T]
	AggregateDeserializer[ID, T]
}

type AggregateRepository[ID aggregate.ID, T aggregate.Root[ID]] struct {
	AggregateSerde AggregateSerde[ID, T]
	MessageSerde   MessageSerde
	DB             *sqlx.DB
}
