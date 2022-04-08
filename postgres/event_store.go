package postgresql

import (
	"github.com/get-eventually/go-eventually/core/event"
	"github.com/get-eventually/go-eventually/core/message"
)

type MessageSerializer interface {
	Serialize(msgType string, eventStreamID event.StreamID, msg message.Message) ([]byte, error)
}

type MessageDeserializer interface {
	Deserialize(msgType string, eventStreamID event.StreamID, data []byte) (message.Message, error)
}

type MessageSerde interface {
	MessageSerializer
	MessageDeserializer
}
