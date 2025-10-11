package uncors

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/charmbracelet/log"
)

func (app *App) internalShutdown(rootCtx context.Context) error {
	app.serversMutex.RLock()
	if len(app.servers) == 0 {
		app.serversMutex.RUnlock()

		return nil
	}
	app.serversMutex.RUnlock()

	app.shuttingDown.Store(true)
	ctx, cancel := context.WithTimeout(rootCtx, shutdownTimeout)
	defer cancel()

	log.Debug("uncors: shutting down all servers ...")

	// Shutdown all servers in parallel
	var waitGroup sync.WaitGroup
	errChan := make(chan error, len(app.servers))

	app.serversMutex.RLock()
	for port, portSrv := range app.servers {
		if portSrv == nil || portSrv.server == nil {
			continue
		}

		waitGroup.Add(1)
		go func(port int, server *http.Server) {
			defer waitGroup.Done()

			log.Debugf("Shutting down server on port %d", port)
			if err := server.Shutdown(ctx); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					log.Errorf("shutdown timeout for server on port %d", port)
				} else {
					log.Errorf("error while shutting down server on port %d: %s", port, err)
				}
				errChan <- err
			}
		}(port, portSrv.server)
	}
	app.serversMutex.RUnlock()

	waitGroup.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	log.Debug("All UNCORS servers closed")

	return nil
}
