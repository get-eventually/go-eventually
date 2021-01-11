package command

import (
	"context"
	"errors"
	"testing"

	"github.com/eventually-rs/eventually-go"
	"github.com/eventually-rs/eventually-go/aggregate"
	"github.com/eventually-rs/eventually-go/eventstore/inmemory"

	"github.com/stretchr/testify/assert"
)

type HandlerScenario struct {
	given []interface{}
}

func Given(events ...interface{}) HandlerScenario { return HandlerScenario{given: events} }

func (sc HandlerScenario) When(command Command) HandlerScenarioWhen {
	return HandlerScenarioWhen{given: sc.given, when: command}
}

func When(command Command) HandlerScenarioWhen { return HandlerScenarioWhen{when: command} }

type HandlerScenarioWhen struct {
	given []interface{}
	when  Command
}

func (sc HandlerScenarioWhen) Then(events ...interface{}) HandlerScenarioThen {
	return HandlerScenarioThen{
		given: sc.given,
		when:  sc.when,
		then:  events,
	}
}

func (sc HandlerScenarioWhen) ThenError(err error) HandlerScenarioThen {
	return HandlerScenarioThen{
		given:     sc.given,
		when:      sc.when,
		wantError: true,
		thenError: err,
	}
}

func (sc HandlerScenarioWhen) ThenFails() HandlerScenarioThen {
	return HandlerScenarioThen{
		given:     sc.given,
		when:      sc.when,
		wantError: true,
	}
}

type HandlerScenarioThen struct {
	given     []interface{}
	when      Command
	then      []interface{}
	thenError error
	wantError bool
}

func (sc HandlerScenarioThen) Using(
	t *testing.T,
	aggregateType aggregate.Type,
	handlerFactory func(*aggregate.Repository) Handler,
) {
	ctx := context.Background()
	store := inmemory.NewEventStore()

	if err := store.Register(ctx, aggregateType.Name(), nil); !assert.NoError(t, err) {
		return
	}

	typedStore, err := store.Type(ctx, aggregateType.Name())
	if !assert.NoError(t, err) {
		return
	}

	if len(sc.given) > 0 {
		_, err = typedStore.
			Instance(sc.when.AggregateID()).
			Append(context.Background(), 0, toEvents(sc.given...)...)

		if !assert.NoError(t, err) {
			return
		}
	}

	trackingStore := &inmemory.TrackingEventStore{Typed: typedStore}
	repository := aggregate.NewRepository(aggregateType, trackingStore)

	handler := handlerFactory(repository)
	err = handler.Handle(context.Background(), sc.when)

	if !sc.wantError {
		assert.NoError(t, err)
		assert.Equal(t, sc.then, toPayloads(trackingStore.Recorded()...))
		return
	}

	if !assert.Error(t, err) {
		return
	}

	if sc.thenError != nil && !assert.True(t, errors.Is(err, sc.thenError)) {
		t.Log("Unexpected error received:", err)
		return
	}
}

func toEvents(payloads ...interface{}) []eventually.Event {
	events := make([]eventually.Event, 0, len(payloads))

	for _, payload := range payloads {
		events = append(events, eventually.Event{Payload: payload})
	}

	return events
}

func toPayloads(events ...eventually.Event) []interface{} {
	payloads := make([]interface{}, 0, len(events))

	for _, event := range events {
		payloads = append(payloads, event.Payload)
	}

	return payloads
}
