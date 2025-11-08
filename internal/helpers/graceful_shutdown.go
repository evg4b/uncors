package helpers

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	notifyFn  = signal.Notify
	sigintFix = func() {
		// fix prints after "^C"
		os.Stdout.WriteString("\n")
	}
)

func GracefulShutdown(ctx context.Context, shutdownFunc func(ctx context.Context) error) {
	if done := waiteSignal(ctx); done {
		return
	}

	err := shutdownFunc(ctx)
	if err != nil {
		panic(err)
	}
}

func waiteSignal(ctx context.Context) bool {
	stop := make(chan os.Signal, 1)

	notifyFn(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	defer close(stop)

	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			sigintFix()
		}
	case <-ctx.Done():
		return true
	}

	return false
}
