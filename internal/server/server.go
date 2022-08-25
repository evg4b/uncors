package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/evg4b/uncors/internal/infrastructure"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	http     http.Server
	httpPort int

	https     http.Server
	httpsPort int
	cert      string
	key       string

	handler func(http.ResponseWriter, *http.Request)
}

const baseAddress = "0.0.0.0"

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
)

const readHeaderTimeout = 30 * time.Second

func NewServer(options ...serverOption) *Server {
	appServer := &Server{
		httpPort:  defaultHTTPPort,
		httpsPort: defaultHTTPSPort,
	}

	for _, option := range options {
		option(appServer)
	}

	address := net.JoinHostPort(baseAddress, strconv.Itoa(appServer.httpPort))
	appServer.http = http.Server{
		ReadHeaderTimeout: readHeaderTimeout,
		Addr:              address,
		Handler: http.HandlerFunc(
			infrastructure.NormalizeHTTPReqDecorator(appServer.handler),
		),
	}

	if appServer.isHTTPSAvialable() {
		address = net.JoinHostPort(baseAddress, strconv.Itoa(appServer.httpsPort))
		appServer.https = http.Server{
			ReadHeaderTimeout: readHeaderTimeout,
			Addr:              address,
			Handler: http.HandlerFunc(
				infrastructure.NormalizeHTTPSReqDecorator(appServer.handler),
			),
		}
	}

	return appServer
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	rungroup, ctx := errgroup.WithContext(ctx)

	rungroup.Go(serveHandler(func() error {
		return s.http.ListenAndServe() // nolint: wrapcheck
	}))

	if s.isHTTPSAvialable() {
		rungroup.Go(serveHandler(func() error {
			return s.https.ListenAndServeTLS(s.cert, s.key) // nolint: wrapcheck
		}))
	}

	shutdownCtx := context.Background()
	rungroup.Go(shutdownHandler(ctx, shutdownCtx, &s.http))
	if s.isHTTPSAvialable() {
		rungroup.Go(shutdownHandler(ctx, shutdownCtx, &s.https))
	}

	if err := rungroup.Wait(); err != nil {
		return fmt.Errorf("uncors server was stopperd with error: %w", err)
	}

	return nil
}

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
