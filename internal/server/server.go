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
	http    Listner
	https   Listner
	cert    string
	key     string
	handler http.Handler
}

type Listner interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
}

const readHeaderTimeout = 30 * time.Second

func NewServer(options ...serverOption) *Server {
	appServer := &Server{}

	for _, option := range options {
		option(appServer)
	}

	return appServer
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	pool, ctx := pool.WithContext(ctx)

	pool.Go(func() error {
		return s.http.ListenAndServe() // nolint: wrapcheck
	})

	if s.isHTTPSAvialable() {
		pool.Go(func() error {
			return s.https.ListenAndServeTLS(s.cert, s.key) // nolint: wrapcheck
		})
	}

	shutdownCtx := context.Background()
	pool.Go(func() error {
		<-ctx.Done()
		var multiError *multierror.Error

		if err := s.http.Shutdown(shutdownCtx); !isSucessClosed(err) {
			multiError = multierror.Append(multiError, err)
		}

		if s.isHTTPSAvialable() {
			if err := s.https.Shutdown(shutdownCtx); !isSucessClosed(err) {
				multiError = multierror.Append(multiError, err)
			}
		}

		return multiError.ErrorOrNil() // nolint: wrapcheck
	})

	if err := pool.Wait(); !isSucessClosed(err) {
		return fmt.Errorf("uncors server was stopperd with error: %w", err)
	}

	return nil
}
