//nolint:wrapcheck
package server

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/log"
	"golang.org/x/net/context"
)

type UncorsServer struct {
	*http.Server
	inShutdown AtomicBool
}

const readHeaderTimeout = 30 * time.Second
const shutdownTimeout = 15 * time.Second

func NewUncorsServer(ctx context.Context, handler http.Handler) *UncorsServer {
	globalCtx, globalCtxCancel := context.WithCancel(ctx)
	server := &http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return globalCtx
		},
		ReadHeaderTimeout: readHeaderTimeout,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			helpers.NormaliseRequest(request)
			handler.ServeHTTP(writer, request)
		}),
		ErrorLog: log.StandardErrorLogAdapter(),
	}
	server.RegisterOnShutdown(globalCtxCancel)

	return &UncorsServer{
		Server: server,
	}
}

func (srv *UncorsServer) ListenAndServe(addr string) error {
	if srv.shuttingDown() {
		return http.ErrServerClosed
	}

	if addr == "" {
		addr = ":http"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer listener.Close()

	srv.Addr = listener.Addr().String()
	err = srv.Serve(listener)
	if err != nil {
		srv.internalShutdown()
	}

	return err
}

func (srv *UncorsServer) ListenAndServeTLS(addr string, certFile, keyFile string) error {
	if srv.shuttingDown() {
		return http.ErrServerClosed
	}

	if addr == "" {
		addr = ":https"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer listener.Close()

	srv.Addr = listener.Addr().String()
	err = srv.ServeTLS(listener, certFile, keyFile)
	if err != nil {
		srv.internalShutdown()
	}

	return err
}

func (srv *UncorsServer) Shutdown(ctx context.Context) error {
	srv.inShutdown.SetTrue()

	return srv.Server.Shutdown(ctx) //nolint:wrapcheck
}

func (srv *UncorsServer) Close() error {
	srv.inShutdown.SetTrue()

	return srv.Server.Close()
}

func (srv *UncorsServer) shuttingDown() bool {
	return srv.inShutdown.IsSet()
}

func (srv *UncorsServer) internalShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	log.Debug("uncors: shutting down ...")
	err := srv.Shutdown(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Errorf("finish: shutdown timeout for UNCORS server")
		} else {
			log.Errorf("finish: error while shutting down UNCORS server: %s", err)
		}
	} else {
		log.Debug("finish: UNCORS server closed")
	}
}
