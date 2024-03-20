package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/get-eventually/go-eventually/internal/user/proto"
	"github.com/get-eventually/go-eventually/serde"
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

	user := &User{ //nolint:exhaustruct // Other fields will be set by eventually.
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
var EventProtoSerde = serde.Fused[*Event, *proto.Event]{
	Serializer:   serde.SerializerFunc[*Event, *proto.Event](protoEventSerializer),
	Deserializer: serde.DeserializerFunc[*Event, *proto.Event](protoEventDeserializer),
}

func protoEventSerializer(evt *Event) (*proto.Event, error) {
	switch kind := evt.Kind.(type) {
	case *WasCreated:
		return &proto.Event{
			Event: &proto.Event_WasCreated_{
				WasCreated: &proto.Event_WasCreated{
					Id:        evt.ID.String(),
					FirstName: kind.FirstName,
					LastName:  kind.LastName,
					BirthDate: timeToDate(kind.BirthDate),
					Email:     kind.Email,
				},
			},
		}, nil
	case *EmailWasUpdated:
		return &proto.Event{
			Event: &proto.Event_EmailWasUpdated_{
				EmailWasUpdated: &proto.Event_EmailWasUpdated{
					Email: kind.Email,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("user.protoEventSerializer: invalid event type, %T", kind)
	}
}

func protoEventDeserializer(evt *proto.Event) (*Event, error) {
	switch t := evt.Event.(type) {
	case *proto.Event_WasCreated_:
		id, err := uuid.Parse(t.WasCreated.Id)
		if err != nil {
			return nil, fmt.Errorf("user.protoEventDeserializer: failed to parse id, %w", err)
		}

		return &Event{
			ID:         id,
			RecordTime: time.Time{},
			Kind: &WasCreated{
				FirstName: t.WasCreated.FirstName,
				LastName:  t.WasCreated.LastName,
				BirthDate: dateToTime(t.WasCreated.BirthDate),
				Email:     t.WasCreated.Email,
			},
		}, nil

	case *proto.Event_EmailWasUpdated_:
		return &Event{
			ID:         uuid.Nil, // FIXME: this should be the actual ID
			RecordTime: time.Time{},
			Kind: &EmailWasUpdated{
				Email: t.EmailWasUpdated.Email,
			},
		}, nil

	default:
		return nil, fmt.Errorf("user.protoEventDeserializer: invalid event type, %T", evt)
	}
}
