package postgres

import (
	"github.com/get-eventually/go-eventually/core/message"
)

type MessageSerializer interface {
	message.Serializer[message.Message, []byte]
}

type MessageDeserializer interface {
	message.Deserializer[[]byte, message.Message]
}

type MessageSerde interface {
	MessageSerializer
	MessageDeserializer
}
