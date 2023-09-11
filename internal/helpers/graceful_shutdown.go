package helpers

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func GracefulShutdown(ctx context.Context, shutdownFunc func(ctx context.Context) error) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			// fix prints after "^C"
			fmt.Println("") // nolint:forbidigo
		}
	case <-ctx.Done():
		return nil
	}

	return shutdownFunc(ctx)
}
