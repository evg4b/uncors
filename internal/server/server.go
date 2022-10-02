package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evg4b/uncors/pkg/pool"
	"github.com/hashicorp/go-multierror"
)

type Server struct {
	http    Listener
	https   Listener
	cert    string
	key     string
	handler http.Handler
}

type Listener interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
}

const readHeaderTimeout = 30 * time.Second

func NewServer(options ...Option) *Server {
	appServer := &Server{}

	for _, option := range options {
		option(appServer)
	}

	return appServer
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	instance, ctx := pool.WithContext(ctx)

	instance.Go(func() error {
		return s.http.ListenAndServe() // nolint: wrapcheck
	})

	if s.isHTTPSAvailable() {
		instance.Go(func() error {
			return s.https.ListenAndServeTLS(s.cert, s.key) // nolint: wrapcheck
		})
	}

	shutdownCtx := context.Background()
	instance.Go(func() error {
		<-ctx.Done()
		var multiError *multierror.Error

		if err := s.http.Shutdown(shutdownCtx); !isSuccessClosed(err) {
			multiError = multierror.Append(multiError, err)
		}

		if s.isHTTPSAvailable() {
			if err := s.https.Shutdown(shutdownCtx); !isSuccessClosed(err) {
				multiError = multierror.Append(multiError, err)
			}
		}

		return multiError.ErrorOrNil() // nolint: wrapcheck
	})

	if err := instance.Wait(); !isSuccessClosed(err) {
		return fmt.Errorf("uncors server was stopperd with error: %w", err)
	}

	return nil
}
