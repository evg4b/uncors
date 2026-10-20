package tracking_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var errHandler = errors.New("handler failed")

type recorder struct {
	mu     sync.Mutex
	events []events.Event
}

func (r *recorder) Publish(event events.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
}

func (r *recorder) all() []events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]events.Event(nil), r.events...)
}

func begin(t *testing.T) (context.Context, *tracking.Scope, *recorder) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.local/users", http.NoBody)
	require.NoError(t, err)

	publisher := &recorder{}
	ctx, scope := tracking.Begin(request.Context(), publisher, 42, request, time.Now())

	return ctx, scope, publisher
}

func started(t *testing.T, event events.Event) events.RequestStarted {
	t.Helper()

	typed, ok := event.(events.RequestStarted)
	require.True(t, ok, "expected RequestStarted, got %T", event)

	return typed
}

func completed(t *testing.T, event events.Event) events.RequestCompleted {
	t.Helper()

	typed, ok := event.(events.RequestCompleted)
	require.True(t, ok, "expected RequestCompleted, got %T", event)

	return typed
}

func data() *contracts.RequestData {
	return &contracts.RequestData{Method: http.MethodGet, Code: http.StatusOK}
}

func TestScope(t *testing.T) {
	t.Run("first claim publishes started with that handler", func(t *testing.T) {
		ctx, _, publisher := begin(t)

		tracking.Claim(ctx, events.HandlerCache)

		published := publisher.all()
		require.Len(t, published, 1)

		event := started(t, published[0])
		assert.Equal(t, uint64(42), event.ID)
		assert.Equal(t, http.MethodGet, event.Method)
		assert.Equal(t, "http://api.local/users", event.URL.String())
		assert.Equal(t, events.HandlerCache, event.Handler)
	})

	t.Run("completed reports the last claim", func(t *testing.T) {
		ctx, scope, publisher := begin(t)

		tracking.Claim(ctx, events.HandlerStatic)
		tracking.Claim(ctx, events.HandlerProxy)
		scope.Finish(data())

		published := publisher.all()
		require.Len(t, published, 2)
		assert.Equal(t, events.HandlerStatic, started(t, published[0]).Handler)

		event := completed(t, published[1])
		assert.Equal(t, uint64(42), event.ID)
		assert.Equal(t, events.HandlerProxy, event.Handler)
		assert.Equal(t, data(), event.Data)
	})

	t.Run("an unclaimed request publishes started with no handler on finish", func(t *testing.T) {
		_, scope, publisher := begin(t)

		scope.Finish(data())

		published := publisher.all()
		require.Len(t, published, 2)
		assert.Equal(t, events.HandlerNone, started(t, published[0]).Handler)
		assert.Equal(t, events.HandlerNone, completed(t, published[1]).Handler)
	})

	t.Run("a rewrite before the claim is reported by both events", func(t *testing.T) {
		ctx, scope, publisher := begin(t)

		tracking.Rewrote(ctx, "/users", "/v2/users")
		tracking.Claim(ctx, events.HandlerProxy)
		scope.Finish(data())

		published := publisher.all()
		rewrite := &events.Rewrite{From: "/users", To: "/v2/users"}
		assert.Equal(t, rewrite, started(t, published[0]).Rewrite)
		assert.Equal(t, rewrite, completed(t, published[1]).Rewrite)
	})

	t.Run("completed carries the handler error", func(t *testing.T) {
		ctx, scope, publisher := begin(t)

		tracking.Claim(ctx, events.HandlerScript)
		tracking.Fail(ctx, errHandler)
		scope.Finish(data())

		assert.ErrorIs(t, completed(t, publisher.all()[1]).Err, errHandler)
	})

	t.Run("reports after finish are ignored", func(t *testing.T) {
		ctx, scope, publisher := begin(t)

		scope.Finish(data())
		scope.Finish(data())
		tracking.Claim(ctx, events.HandlerProxy)

		assert.Len(t, publisher.all(), 2)
	})

	t.Run("a context without a scope is ignored", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tracking.Claim(t.Context(), events.HandlerProxy)
			tracking.Rewrote(t.Context(), "/a", "/b")
			tracking.Fail(t.Context(), errHandler)
		})
	})

	t.Run("concurrent claims publish exactly one started before completed", func(t *testing.T) {
		ctx, scope, publisher := begin(t)

		var waitGroup sync.WaitGroup

		for range 16 {
			waitGroup.Go(func() { tracking.Claim(ctx, events.HandlerProxy) })
		}

		waitGroup.Wait()
		scope.Finish(data())

		published := publisher.all()
		require.Len(t, published, 2)
		assert.IsType(t, events.RequestStarted{}, published[0])
		assert.IsType(t, events.RequestCompleted{}, published[1])
	})
}

func TestOriginRejection(t *testing.T) {
	t.Run("is not rejected by default", func(t *testing.T) {
		ctx, _, _ := begin(t)

		assert.False(t, tracking.OriginRejected(ctx))
	})

	t.Run("is rejected for the whole request", func(t *testing.T) {
		ctx, _, _ := begin(t)

		childCtx, child := telemetry.Start(ctx, "router.route")
		tracking.RejectOrigin(childCtx)
		child.End()

		assert.True(t, tracking.OriginRejected(ctx))
		assert.True(t, tracking.OriginRejected(childCtx))
	})

	t.Run("is ignored without a scope", func(t *testing.T) {
		tracking.RejectOrigin(t.Context())

		assert.False(t, tracking.OriginRejected(t.Context()))
	})
}

func TestScopeRecordsOnTheRootSpan(t *testing.T) {
	recorded := testutils.Telemetry(t)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.local/users", http.NoBody)
	require.NoError(t, err)

	rootCtx, root := telemetry.StartServer(request.Context(), "HTTP GET")
	ctx, scope := tracking.Begin(rootCtx, &recorder{}, 42, request, time.Now())

	childCtx, child := telemetry.Start(ctx, "mock.serve")
	tracking.Claim(childCtx, events.HandlerMock)
	tracking.Rewrote(ctx, "/users", "/v2/users")
	tracking.Fail(ctx, errHandler)
	tracking.RejectOrigin(childCtx)
	child.End()
	root.End()

	assert.Equal(t, events.HandlerMock, scope.Handler())

	ended := recorded.Spans.Ended()
	require.Len(t, ended, 2)

	claimed, rootSpan := ended[0], ended[1]
	require.Len(t, claimed.Events(), 1)
	assert.Equal(t, "claimed", claimed.Events()[0].Name)

	attrs := rootSpan.Attributes()
	assert.Contains(t, attrs, attribute.String("uncors.handler", events.HandlerMock.String()))
	assert.Contains(t, attrs, attribute.String("uncors.rewrite.from", "/users"))
	assert.Contains(t, attrs, attribute.String("uncors.rewrite.to", "/v2/users"))
	assert.Contains(t, attrs, attribute.Bool("uncors.origin_rejected", true))
	assert.Equal(t, codes.Error, rootSpan.Status().Code)
	assert.Equal(t, errHandler.Error(), rootSpan.Status().Description)
}
