package uncors

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

type serveConfig struct {
	addr        string
	serve       func(l net.Listener) error
	setListener func(l net.Listener)
}

func (app *App) listenAndServeForPort(ctx context.Context, srv *portServer, addr string) error {
	return app.internalServeForPort(ctx, &serveConfig{
		addr:  addr,
		serve: srv.server.Serve,
		setListener: func(l net.Listener) {
			srv.mutex.Lock()
			srv.listener = l
			srv.mutex.Unlock()
		},
	})
}

func (app *App) listenAndServeTLSForPort(ctx context.Context, srv *portServer, addr string, config *tls.Config) error {
	return app.internalServeForPort(ctx, &serveConfig{
		addr: addr,
		serve: func(l net.Listener) error {
			tlsListener := tls.NewListener(l, config)

			return srv.server.Serve(tlsListener)
		},
		setListener: func(l net.Listener) {
			srv.mutex.Lock()
			srv.listener = l
			srv.mutex.Unlock()
		},
	})
}

func (app *App) internalServeForPort(ctx context.Context, config *serveConfig) error {
	if app.shuttingDown.Load() {
		return http.ErrServerClosed
	}

	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "tcp", config.addr)
	if err != nil {
		return err
	}

	config.setListener(listener)

	defer func() { config.setListener(nil) }()

	err = config.serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		shutdownError := app.internalShutdown(context.TODO())
		if shutdownError != nil && !errors.Is(shutdownError, http.ErrServerClosed) {
			panic(shutdownError)
		}

		return err
	}

	return nil
}
