package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/version"
)

const (
	versionCheckDelay = 50 * time.Millisecond
	frontendBuffer    = 1000
	commandsBuffer    = 8
)

type supervisor struct {
	env         Env
	cfgPath     string
	interactive bool
	bus         *events.Bus
	engine      *engine.Engine

	commands chan events.Command
	reloads  chan struct{}
}

func supervise(
	ctx context.Context,
	env Env,
	uncorsConfig *config.UncorsConfig,
	cfgPath string,
	frontend frontend,
) error {
	bus := events.NewBus()
	supervisor := &supervisor{
		env:         env,
		cfgPath:     cfgPath,
		interactive: uncorsConfig.Interactive,
		bus:         bus,
		engine:      engine.New(env.Fs, engine.WithPublisher(bus)),
		commands:    make(chan events.Command, commandsBuffer),
		// One pending reload is enough: it reads the file as it is then.
		reloads: make(chan struct{}, 1),
	}

	subscription := bus.Subscribe(frontendBuffer)
	frontendDone := make(chan error, 1)

	telemetry.OnExportError(func(signal string, err error) {
		bus.Publish(events.Notice{
			Level: events.NoticeWarning,
			Text:  "Tracing is enabled but " + signal + " cannot be exported: " + err.Error(),
		})
	})

	go func() {
		frontendDone <- frontend(subscription, supervisor.commands)
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := supervisor.engine.Start(ctx, uncorsConfig)
	if err != nil {
		supervisor.announceStopped()
		<-frontendDone

		return errors.Join(err, supervisor.engine.Close())
	}

	if telemetry.Enabled() {
		bus.Publish(events.Notice{
			Level: events.NoticeInfo,
			Text:  "Tracing is enabled, exporting to " + telemetry.Endpoint(),
		})
	}

	go supervisor.checkVersion(ctx, uncorsConfig.Proxy)

	supervisor.watchConfig(ctx)

	frontendExited, frontendErr := supervisor.run(ctx, frontendDone)

	supervisor.shutdown(ctx)
	supervisor.announceStopped()

	if !frontendExited {
		frontendErr = <-frontendDone
	}

	return frontendErr
}

func (s *supervisor) run(ctx context.Context, frontendDone <-chan error) (bool, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	defer signal.Stop(signals)

	for {
		select {
		case <-ctx.Done():
			return false, nil
		case sig := <-signals:
			s.onSignal(sig)

			return false, nil
		case err := <-frontendDone:
			return true, err
		case <-s.reloads:
			s.reload(ctx)
		case command := <-s.commands:
			if s.handle(ctx, command) {
				return false, nil
			}
		}
	}
}

func (s *supervisor) handle(ctx context.Context, command events.Command) bool {
	switch command.(type) {
	case events.Restart:
		s.reload(ctx)
	case events.Shutdown:
		return true
	}

	return false
}

func (s *supervisor) onSignal(sig os.Signal) {
	telemetry.Info(context.Background(), "shutdown signal received", slog.String("signal", sig.String()))

	if sig == syscall.SIGINT && !s.interactive {
		// Move past the "^C" the terminal echoed.
		_, _ = io.WriteString(s.env.Stdout, "\n")
	}
}

func (s *supervisor) reload(ctx context.Context) {
	ctx, span := telemetry.Start(ctx, "supervisor.reload")
	defer span.End()

	uncorsConfig, _, err := config.LoadConfiguration(ctx, s.env.Fs, nil, s.env.Args)
	if err == nil {
		err = s.engine.Restart(ctx, uncorsConfig)
	}

	telemetry.ConfigReloaded(ctx, err)

	if err != nil {
		telemetry.Fail(span, err)
		s.bus.Publish(events.ReloadFailed{Err: err})
	}
}

// Uses a fresh context: ctx may be the one that ended the run.
func (s *supervisor) shutdown(ctx context.Context) {
	ctx, span := telemetry.Start(context.WithoutCancel(ctx), "supervisor.shutdown")
	defer span.End()

	err := s.engine.Shutdown(ctx)
	if err != nil {
		telemetry.Fail(span, err)
		telemetry.Warn(ctx, "in-flight requests did not finish before shutdown", telemetry.Err(err))
	}

	s.engine.Wait()
}

func (s *supervisor) announceStopped() {
	s.bus.Publish(events.ServerStopped{})
	s.bus.Close()
}

func (s *supervisor) watchConfig(ctx context.Context) {
	if s.cfgPath == "" {
		return
	}

	watcher := config.NewWatcher(s.cfgPath)

	err := watcher.Watch(ctx, func() {
		select {
		case s.reloads <- struct{}{}:
		default:
		}
	}, func(err error) {
		telemetry.Error(ctx, "config watcher error", telemetry.Err(err))
		s.bus.Publish(events.Notice{Level: events.NoticeError, Text: "Config watcher error: " + err.Error()})
	})
	if err != nil {
		telemetry.Error(ctx, "failed to watch config file",
			slog.String("path", s.cfgPath), telemetry.Err(err))
		s.bus.Publish(events.Notice{Level: events.NoticeError, Text: "Failed to watch config file: " + err.Error()})

		return
	}

	go func() {
		<-ctx.Done()

		_ = watcher.Close()
	}()
}

func (s *supervisor) checkVersion(ctx context.Context, proxyURL string) {
	select {
	case <-time.After(versionCheckDelay):
	case <-ctx.Done():
		return
	}

	checker := version.NewVersionChecker(
		version.WithHTTPClient(infra.MakeHTTPClient(proxyURL)),
		version.WithCurrentVersion(s.env.Version),
	)

	latest, available := checker.CheckNewVersion(ctx)
	if available {
		s.bus.Publish(events.NewVersionAvailable{Current: checker.Current(), Latest: latest})
	}
}
