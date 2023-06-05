package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/get-eventually/go-eventually/core/aggregate"
	"github.com/get-eventually/go-eventually/core/query"
	"github.com/get-eventually/go-eventually/core/version"
)

var ErrEmptyID = errors.New("user: empty id provided")

type ViewModel struct {
	Version             version.Version
	ID                  uuid.UUID
	FirstName, LastName string
	BirthDate           time.Time
	Email               string
}

func buildViewModel(u *User) ViewModel {
	return ViewModel{
		Version:   u.Version(),
		ID:        u.id,
		FirstName: u.firstName,
		LastName:  u.lastName,
		BirthDate: u.birthDate,
		Email:     u.email,
	}
}

type GetQuery struct {
	ID uuid.UUID
}

func (GetQuery) Name() string { return "GetUser" }

type GetQueryHandler struct {
	Repository aggregate.Getter[uuid.UUID, *User]
}

func (h GetQueryHandler) Handle(ctx context.Context, q query.Envelope[GetQuery]) (ViewModel, error) {
	makeError := func(err error) error {
		return fmt.Errorf("user.GetQuery: failed to handle query, %w", err)
	}

	if q.Message.ID == uuid.Nil {
		return ViewModel{}, makeError(ErrEmptyID)
	}

	user, err := h.Repository.Get(ctx, q.Message.ID)
	if err != nil {
		return ViewModel{}, makeError(err)
	}

	return buildViewModel(user), nil
}
