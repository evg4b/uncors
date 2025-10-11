package uncors

import (
	"context"
	"errors"
	"net"
	"net/http"
)

type serveConfig struct {
	addr        string
	serve       func(l net.Listener) error
	setListener func(l net.Listener)
}

func (app *App) listenAndServeForPort(ps *portServer, addr string) error {
	return app.internalServeForPort(ps, &serveConfig{
		addr:  addr,
		serve: ps.server.Serve,
		setListener: func(l net.Listener) {
			ps.listener = l
		},
	})
}

func (app *App) listenAndServeTLSForPort(ps *portServer, addr string, certFile, keyFile string) error {
	return app.internalServeForPort(ps, &serveConfig{
		addr: addr,
		serve: func(l net.Listener) error {
			return ps.server.ServeTLS(l, certFile, keyFile)
		},
		setListener: func(l net.Listener) {
			ps.listener = l
		},
	})
}

func (app *App) internalServeForPort(ps *portServer, config *serveConfig) error {
	if app.shuttingDown.Load() {
		return http.ErrServerClosed
	}

	listener, err := net.Listen("tcp", config.addr)
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
