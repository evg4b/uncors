package uncors

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
)

func (app *App) internalShutdown(rootCtx context.Context) error {
	if app.server == nil {
		return nil
	}

	app.shuttingDown.Store(true)
	ctx, cancel := context.WithTimeout(rootCtx, shutdownTimeout)
	defer cancel()

	log.Debug("uncors: shutting down ...")

	if err := app.server.Shutdown(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Errorf("shutdown timeout for UNCORS server")
		} else {
			log.Errorf("error while shutting down UNCORS server: %s", err)
		}

		return err
	}

	log.Debug("UNCORS server closed")

	return nil
}
