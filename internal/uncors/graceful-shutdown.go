package uncors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/log"
)

func GracefulShutdown(ctx context.Context, timeout time.Duration, action func(ctx context.Context) error) {
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
	stop := make(chan os.Signal, len(signals))
	signal.Notify(stop, signals...)

	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			// fix prints after "^C"
			fmt.Println("") // nolint:forbidigo
		}
	case <-ctx.Done():
		return
	}

	log.Debug("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log.Debug("shutting down application ...")

	if err := action(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Error("shutdown timeout")
		} else {
			log.Errorf("error while shutting down %s", err)
		}
	} else {
		log.Debug("application closed")
	}
}
