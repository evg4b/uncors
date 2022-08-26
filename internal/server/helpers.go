package server

import (
	"context"
	"errors"
	"net/http"
)

func shutdownHandler(ctx, shutdownCtx context.Context, server *http.Server) func() error {
	return func() error {
		<-ctx.Done()

		if err := server.Shutdown(shutdownCtx); !isSucessClosed(err) {
			return err // nolint: wrapcheck
		}

		return nil
	}
}

func serveHandler(handler func() error) func() error {
	return func() error {
		if err := handler(); !isSucessClosed(err) {
			return err // nolint: wrapcheck
		}

		return nil
	}
}

func (s *Server) isHTTPSAvialable() bool {
	return len(s.cert) > 0 && len(s.key) > 0
}

func isSucessClosed(err error) bool {
	return err == nil || errors.Is(err, http.ErrServerClosed)
}
