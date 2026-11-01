package infra

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
)

var ErrResponseNotCasted = errors.New("received incorrect response writer type")

type HandlerFunc func(contracts.ResponseWriter, *contracts.Request) error

func (f HandlerFunc) ServeHTTP(w contracts.ResponseWriter, r *contracts.Request) error {
	return f(w, r)
}

func CastToHTTPHandler(handler contracts.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writer, ok := response.(contracts.ResponseWriter)
		if !ok {
			panic(ErrResponseNotCasted)
		}

		err := handler.ServeHTTP(writer, request)
		if err != nil {
			logFailure(request.Context(), request, err)
			tracking.Fail(request.Context(), err)
			HTTPError(writer, request, err)
		}
	})
}

// logFailure records the error a handler returned. Every handler error ends
// here, so handlers do not log the errors they return.
func logFailure(ctx context.Context, request *http.Request, err error) {
	if !telemetry.Enabled() {
		return
	}

	attrs := []slog.Attr{
		slog.String("method", request.Method),
		slog.String("url", helpers.LogURL(request.URL)),
		telemetry.Err(err),
	}

	if ctx.Err() != nil {
		telemetry.Debug(ctx, "request cancelled by the client", attrs...)

		return
	}

	telemetry.Error(ctx, "request failed", attrs...)
}
