package testutils

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"
)

func RejectOrigin(t *testing.T, request *http.Request) *http.Request {
	t.Helper()

	ctx, _ := tracking.Begin(request.Context(), events.NoopPublisher{}, 1, request, time.Now())
	tracking.RejectOrigin(ctx)

	return request.WithContext(ctx)
}
