package router

import (
	"log/slog"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/go-http-utils/headers"
)

func checkOrigin(origins config.AllowedOrigins, handler contracts.Handler) contracts.Handler {
	if len(origins) == 0 {
		return handler
	}

	return infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		if origin := request.Header.Get(headers.Origin); !origins.Allows(origin) {
			tracking.RejectOrigin(request.Context())

			if telemetry.Enabled() {
				telemetry.Debug(request.Context(), "origin is not allowed, CORS headers withheld",
					slog.String("origin", origin))
			}
		}

		return handler.ServeHTTP(writer, request)
	})
}
