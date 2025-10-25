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

func (app *App) listenAndServeForPort(portSrv *portServer, addr string) error {
	return app.internalServeForPort(portSrv, &serveConfig{
		addr:  addr,
		serve: portSrv.server.Serve,
		setListener: func(l net.Listener) {
			portSrv.mutex.Lock()
			portSrv.listener = l
			portSrv.mutex.Unlock()
		},
	})
}

func (app *App) listenAndServeTLSForPort(portSrv *portServer, addr string, tlsConfig *tls.Config) error {
	return app.internalServeForPort(portSrv, &serveConfig{
		addr: addr,
		serve: func(l net.Listener) error {
			tlsListener := tls.NewListener(l, tlsConfig)

			return portSrv.server.Serve(tlsListener)
		},
		setListener: func(l net.Listener) {
			portSrv.mutex.Lock()
			portSrv.listener = l
			portSrv.mutex.Unlock()
		},
	})
}

func (app *App) internalServeForPort(_ *portServer, config *serveConfig) error {
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
