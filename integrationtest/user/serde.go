package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/get-eventually/go-eventually/core/message"
	"github.com/get-eventually/go-eventually/core/serde"
	"github.com/get-eventually/go-eventually/integrationtest/user/proto"
)

func timeToDate(t time.Time) *date.Date {
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

func dateToTime(d *date.Date) time.Time {
	return time.Date(
		int(d.Year), time.Month(d.Month), int(d.Day),
		0, 0, 0, 0, time.UTC,
	)
}

// ProtoSerde is the serde.Serde implementation for a User to map
// to its Protobuf type, defined in the proto/ folder.
var ProtoSerde = serde.Fused[*User, *proto.User]{
	Serializer:   serde.SerializerFunc[*User, *proto.User](protoSerializer),
	Deserializer: serde.DeserializerFunc[*User, *proto.User](protoDeserializer),
}

func protoSerializer(user *User) (*proto.User, error) {
	return &proto.User{
		Id:        user.id.String(),
		FirstName: user.firstName,
		LastName:  user.lastName,
		BirthDate: timeToDate(user.birthDate),
		Email:     user.email,
	}, nil
}

func protoDeserializer(src *proto.User) (*User, error) {
	id, err := uuid.Parse(src.Id)
	if err != nil {
		return nil, fmt.Errorf("user.protoDeserialize: failed to deserialize user id, %w", err)
	}

	user := &User{
		id:        id,
		firstName: src.FirstName,
		lastName:  src.LastName,
		birthDate: dateToTime(src.BirthDate),
		email:     src.Email,
	}

	return user, nil
}

// EventProtoSerde is the serde.Serde implementation for User domain events
// to map to their Protobuf type, defined in the proto/ folder.
var EventProtoSerde = serde.Fused[message.Message, *proto.Event]{
	Serializer:   serde.SerializerFunc[message.Message, *proto.Event](protoEventSerializer),
	Deserializer: serde.DeserializerFunc[message.Message, *proto.Event](protoEventDeserializer),
}

func protoEventSerializer(msg message.Message) (*proto.Event, error) {
	switch evt := msg.(type) {
	case WasCreated:
		return &proto.Event{
			Event: &proto.Event_WasCreated_{
				WasCreated: &proto.Event_WasCreated{
					Id:        evt.ID.String(),
					FirstName: evt.FirstName,
					LastName:  evt.LastName,
					BirthDate: timeToDate(evt.BirthDate),
					Email:     evt.Email,
				},
			},
		}, nil

	case EmailWasUpdated:
		return &proto.Event{
			Event: &proto.Event_EmailWasUpdated_{
				EmailWasUpdated: &proto.Event_EmailWasUpdated{
					Email: evt.Email,
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("user.protoEventSerializer: invalid event type, %T", msg)
	}
}

func protoEventDeserializer(evt *proto.Event) (message.Message, error) {
	switch t := evt.Event.(type) {
	case *proto.Event_WasCreated_:
		id, err := uuid.Parse(t.WasCreated.Id)
		if err != nil {
			return nil, fmt.Errorf("user.protoEventDeserializer: failed to parse id, %w", err)
		}

		return WasCreated{
			ID:        id,
			FirstName: t.WasCreated.FirstName,
			LastName:  t.WasCreated.LastName,
			BirthDate: dateToTime(t.WasCreated.BirthDate),
			Email:     t.WasCreated.Email,
		}, nil

	case *proto.Event_EmailWasUpdated_:
		return EmailWasUpdated{
			Email: t.EmailWasUpdated.Email,
		}, nil

	default:
		return nil, fmt.Errorf("user.protoEventDeserializer: invalid event type, %T", evt)
	}
}
