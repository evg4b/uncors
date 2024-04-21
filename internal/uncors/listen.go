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

func (app *App) listenAndServe(addr string) error {
	return app.internalServe(&serveConfig{
		addr:  addr,
		serve: app.server.Serve,
		setListener: func(l net.Listener) {
			app.httpListenerMutex.Lock()
			defer app.httpListenerMutex.Unlock()
			app.httpListener = l
		},
	})
}

func (app *App) listenAndServeTLS(addr string, certFile, keyFile string) error {
	return app.internalServe(&serveConfig{
		addr: addr,
		serve: func(l net.Listener) error {
			return app.server.ServeTLS(l, certFile, keyFile)
		},
		setListener: func(l net.Listener) {
			app.httpsListenerMutex.Lock()
			defer app.httpsListenerMutex.Unlock()
			app.httpsListener = l
		},
	})
}

func (app *App) internalServe(config *serveConfig) error {
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
