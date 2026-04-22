package opentelemetry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"github.com/get-eventually/go-eventually/aggregate"
	"github.com/get-eventually/go-eventually/event"
	"github.com/get-eventually/go-eventually/internal/user"
	otelex "github.com/get-eventually/go-eventually/opentelemetry"
)

var testBirthDate = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

func newUserRepository() aggregate.Repository[uuid.UUID, *user.User] {
	return aggregate.NewEventSourcedRepository(event.NewInMemoryStore(), user.Type)
}

func newTestUser(t *testing.T, id uuid.UUID) *user.User {
	t.Helper()

	usr, err := user.Create(id, "Jane", "Doe", "jane@doe.com", testBirthDate, time.Now())
	require.NoError(t, err)

	return usr
}

func TestNewInstrumentedRepository_Succeeds(t *testing.T) {
	h := newHarness(t)

	repo, err := otelex.NewInstrumentedRepository(user.Type, newUserRepository(), h.options()...)
	require.NoError(t, err)
	require.NotNil(t, repo)
}

func TestGet_DelegatesAndReturnsRoot(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)

	require.NoError(t, repo.Save(t.Context(), usr))

	got, err := repo.Get(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, usr, got)
}

func TestGet_RecordsSpan_WithVersionAttributeOnSuccess(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	id := newUUID(t)
	usr := newTestUser(t, id)
	require.NoError(t, inner.Save(t.Context(), usr))

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	_, err = repo.Get(t.Context(), id)
	require.NoError(t, err)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, otelex.RepositoryGetSpanName, span.Name())
	assert.Equal(t, codes.Unset, span.Status().Code)
	assert.Contains(t, span.Attributes(), otelex.AggregateTypeAttribute.String(user.Type.Name))
	assert.Contains(t, span.Attributes(), otelex.AggregateIDAttribute.String(id.String()))
	assert.Contains(t, span.Attributes(), otelex.AggregateVersionAttribute.Int64(int64(usr.Version())))
}

func TestGet_RecordsHistogram_WithAggregateTypeAttribute(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	id := newUUID(t)
	usr := newTestUser(t, id)
	require.NoError(t, inner.Save(t.Context(), usr))

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	_, err = repo.Get(t.Context(), id)
	require.NoError(t, err)

	sm := h.collectScopeMetrics(t)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{Name: otelex.InstrumentationName},
		Metrics: []metricdata.Metrics{
			{
				Name:        otelex.RepositoryGetDurationMetricName,
				Description: otelex.RepositoryGetDurationMetricDescription,
				Unit:        otelex.MetricUnitMilliseconds,
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								otelex.AggregateTypeAttribute.String(user.Type.Name),
								otelex.ErrorAttribute.Bool(false),
							),
						},
					},
				},
			},
		},
	}

	got := metricdata.ScopeMetrics{
		Scope:   sm.Scope,
		Metrics: []metricdata.Metrics{findMetric(t, &sm, otelex.RepositoryGetDurationMetricName)},
	}

	metricdatatest.AssertEqual(t, want, got,
		metricdatatest.IgnoreTimestamp(),
		metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreExemplars(),
	)
}

func TestGet_Error_RecordsErrorOnSpanAndMetric(t *testing.T) {
	h := newHarness(t)

	inner := &errorRepository[uuid.UUID, *user.User]{getErr: aggregate.ErrRootNotFound}

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)

	_, err = repo.Get(t.Context(), id)
	require.ErrorIs(t, err, aggregate.ErrRootNotFound)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, aggregate.ErrRootNotFound.Error(), span.Status().Description)
	assert.True(t, hasExceptionEvent(span), "span should have recorded an exception event")

	// aggregate.version should NOT be present on the span when Get fails.
	for _, attr := range span.Attributes() {
		assert.NotEqual(t, otelex.AggregateVersionAttribute, attr.Key,
			"aggregate.version should not be set on the span when Get fails")
	}

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, otelex.RepositoryGetDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 1)

	val, ok := data.DataPoints[0].Attributes.Value(otelex.ErrorAttribute)
	require.True(t, ok)
	assert.True(t, val.AsBool(), "error attribute should be true on failed Get")
}

func TestGet_SpanIsChildOfParent(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	id := newUUID(t)
	usr := newTestUser(t, id)
	require.NoError(t, inner.Save(t.Context(), usr))

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	ctx, parent := h.tracer.Tracer("test").Start(t.Context(), "parent")

	_, err = repo.Get(ctx, id)
	require.NoError(t, err)

	parent.End()

	spans := h.endedSpans()
	require.Len(t, spans, 2)

	getSpan, parentSpan := spans[0], spans[1]
	if getSpan.Name() != otelex.RepositoryGetSpanName {
		getSpan, parentSpan = spans[1], spans[0]
	}

	assert.Equal(t, otelex.RepositoryGetSpanName, getSpan.Name())
	assert.Equal(t, "parent", parentSpan.Name())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), getSpan.Parent().SpanID())
}

func TestSave_DelegatesAndPersistsRoot(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)

	require.NoError(t, repo.Save(t.Context(), usr))

	got, err := inner.Get(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, usr.AggregateID(), got.AggregateID())
}

func TestSave_RecordsSpan(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)

	require.NoError(t, repo.Save(t.Context(), usr))

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, otelex.RepositorySaveSpanName, span.Name())
	assert.Equal(t, codes.Unset, span.Status().Code)
	assert.Contains(t, span.Attributes(), otelex.AggregateTypeAttribute.String(user.Type.Name))
	assert.Contains(t, span.Attributes(), otelex.AggregateIDAttribute.String(id.String()))
	assert.Contains(t, span.Attributes(), otelex.AggregateVersionAttribute.Int64(int64(usr.Version())))
}

func TestSave_RecordsHistogram(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)
	require.NoError(t, repo.Save(t.Context(), usr))

	sm := h.collectScopeMetrics(t)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{Name: otelex.InstrumentationName},
		Metrics: []metricdata.Metrics{
			{
				Name:        otelex.RepositorySaveDurationMetricName,
				Description: otelex.RepositorySaveDurationMetricDescription,
				Unit:        otelex.MetricUnitMilliseconds,
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								otelex.AggregateTypeAttribute.String(user.Type.Name),
								otelex.ErrorAttribute.Bool(false),
							),
						},
					},
				},
			},
		},
	}

	got := metricdata.ScopeMetrics{
		Scope:   sm.Scope,
		Metrics: []metricdata.Metrics{findMetric(t, &sm, otelex.RepositorySaveDurationMetricName)},
	}

	metricdatatest.AssertEqual(t, want, got,
		metricdatatest.IgnoreTimestamp(),
		metricdatatest.IgnoreValue(),
		metricdatatest.IgnoreExemplars(),
	)
}

func TestSave_Error_RecordsErrorOnSpanAndMetric(t *testing.T) {
	h := newHarness(t)

	wantErr := errors.New("save failed")
	inner := &errorRepository[uuid.UUID, *user.User]{saveErr: wantErr}

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)

	err = repo.Save(t.Context(), usr)
	require.ErrorIs(t, err, wantErr)

	spans := h.endedSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, wantErr.Error(), span.Status().Description)
	assert.True(t, hasExceptionEvent(span))

	sm := h.collectScopeMetrics(t)
	m := findMetric(t, &sm, otelex.RepositorySaveDurationMetricName)

	data, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, data.DataPoints, 1)

	val, ok := data.DataPoints[0].Attributes.Value(otelex.ErrorAttribute)
	require.True(t, ok)
	assert.True(t, val.AsBool())
}

func TestSave_SpanIsChildOfParent(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	id := newUUID(t)
	usr := newTestUser(t, id)

	ctx, parent := h.tracer.Tracer("test").Start(t.Context(), "parent")

	require.NoError(t, repo.Save(ctx, usr))

	parent.End()

	spans := h.endedSpans()
	require.Len(t, spans, 2)

	saveSpan, parentSpan := spans[0], spans[1]
	if saveSpan.Name() != otelex.RepositorySaveSpanName {
		saveSpan, parentSpan = spans[1], spans[0]
	}

	assert.Equal(t, otelex.RepositorySaveSpanName, saveSpan.Name())
	assert.Equal(t, "parent", parentSpan.Name())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), saveSpan.Parent().SpanID())
}

func TestRepository_AttributeSlicesDoNotAlias(t *testing.T) {
	h := newHarness(t)

	inner := newUserRepository()

	id1 := newUUID(t)
	id2 := newUUID(t)

	usr1, err := user.Create(id1, "Jane", "Doe", "jane@doe.com", testBirthDate, time.Now())
	require.NoError(t, err)
	require.NoError(t, inner.Save(t.Context(), usr1))

	usr2, err := user.Create(id2, "John", "Doe", "john@doe.com", testBirthDate, time.Now())
	require.NoError(t, err)
	require.NoError(t, inner.Save(t.Context(), usr2))

	repo, err := otelex.NewInstrumentedRepository(user.Type, inner, h.options()...)
	require.NoError(t, err)

	_, err = repo.Get(t.Context(), id1)
	require.NoError(t, err)

	_, err = repo.Get(t.Context(), id2)
	require.NoError(t, err)

	spans := h.endedSpans()
	require.Len(t, spans, 2)

	assert.Contains(t, spans[0].Attributes(), otelex.AggregateIDAttribute.String(id1.String()))
	assert.NotContains(t, spans[0].Attributes(), otelex.AggregateIDAttribute.String(id2.String()))

	assert.Contains(t, spans[1].Attributes(), otelex.AggregateIDAttribute.String(id2.String()))
	assert.NotContains(t, spans[1].Attributes(), otelex.AggregateIDAttribute.String(id1.String()))
}
