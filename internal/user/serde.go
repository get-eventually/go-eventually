package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/get-eventually/go-eventually/internal/user/gen/user/v1"
	"github.com/get-eventually/go-eventually/message"
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
var ProtoSerde = serde.Fused[*User, *userv1.User]{
	Serializer:   serde.SerializerFunc[*User, *userv1.User](protoSerializer),
	Deserializer: serde.DeserializerFunc[*User, *userv1.User](protoDeserializer),
}

func protoSerializer(user *User) (*userv1.User, error) {
	return &userv1.User{
		Id:        user.id.String(),
		FirstName: user.firstName,
		LastName:  user.lastName,
		BirthDate: timeToDate(user.birthDate),
		Email:     user.email,
	}, nil
}

func protoDeserializer(src *userv1.User) (*User, error) {
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
var EventProtoSerde = serde.Fused[message.Message, *userv1.Event]{
	Serializer:   serde.SerializerFunc[message.Message, *userv1.Event](protoEventSerializer),
	Deserializer: serde.DeserializerFunc[message.Message, *userv1.Event](protoEventDeserializer),
}

func protoEventSerializer(evt message.Message) (*userv1.Event, error) {
	userEvent, ok := evt.(*Event)
	if !ok {
		return nil, fmt.Errorf("user.protoEventSerializer: unexpected event type, %T", evt)
	}

	switch kind := userEvent.Kind.(type) {
	case *WasCreated:
		return &userv1.Event{
			Id:         userEvent.ID.String(),
			RecordTime: timestamppb.New(userEvent.RecordTime),
			Kind: &userv1.Event_WasCreated_{
				WasCreated: &userv1.Event_WasCreated{
					FirstName: kind.FirstName,
					LastName:  kind.LastName,
					BirthDate: timeToDate(kind.BirthDate),
					Email:     kind.Email,
				},
			},
		}, nil
	case *EmailWasUpdated:
		return &userv1.Event{
			Id:         userEvent.ID.String(),
			RecordTime: timestamppb.New(userEvent.RecordTime),
			Kind: &userv1.Event_EmailWasUpdated_{
				EmailWasUpdated: &userv1.Event_EmailWasUpdated{
					Email: kind.Email,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("user.protoEventSerializer: unexpected event kind type, %T", kind)
	}
}

func protoEventDeserializer(evt *userv1.Event) (message.Message, error) {
	id, err := uuid.Parse(evt.Id)
	if err != nil {
		return nil, fmt.Errorf("user.protoEventDeserializer: failed to parse id, %w", err)
	}

	switch t := evt.Kind.(type) {
	case *userv1.Event_WasCreated_:
		return &Event{
			ID:         id,
			RecordTime: evt.RecordTime.AsTime(),
			Kind: &WasCreated{
				FirstName: t.WasCreated.FirstName,
				LastName:  t.WasCreated.LastName,
				BirthDate: dateToTime(t.WasCreated.BirthDate),
				Email:     t.WasCreated.Email,
			},
		}, nil

	case *userv1.Event_EmailWasUpdated_:
		return &Event{
			ID:         id,
			RecordTime: evt.RecordTime.AsTime(),
			Kind: &EmailWasUpdated{
				Email: t.EmailWasUpdated.Email,
			},
		}, nil

	default:
		return nil, fmt.Errorf("user.protoEventDeserializer: invalid event type, %T", evt)
	}
}
