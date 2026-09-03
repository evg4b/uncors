// Package render turns application events into console output.
//
// It is the only place that decides how a service event looks. Both run modes
// use it, so neither can drift from the other in what it reports; the service
// itself renders nothing.
package render

import (
	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
)

// Renderer writes service events to a contracts.Output.
type Renderer struct {
	output  contracts.Output
	version string
}

func New(output contracts.Output, version string) *Renderer {
	return &Renderer{output: output, version: version}
}

// Consume renders every event on the stream and returns when it closes.
func (r *Renderer) Consume(events <-chan app.Event) {
	for event := range events {
		r.Render(event)
	}
}

// Render writes a single event.
func (r *Renderer) Render(event app.Event) {
	switch typed := event.(type) {
	case app.LifecycleEvent:
		r.lifecycle(typed)
	case app.LogEvent:
		r.log(typed)
	}
}

func (r *Renderer) lifecycle(event app.LifecycleEvent) {
	switch event.State {
	case app.StateStarting:
		r.banner(event)
	case app.StateStarted:
	case app.StateStartFailed:
		r.output.Errorf("Failed to start server: %v", event.Err)
	case app.StateReloading:
		r.output.Info("Restarting server....")
	case app.StateReloaded:
		r.output.InfoBox("Server restarted", event.Mappings.String())
	case app.StateReloadFailed:
		r.output.Errorf("Failed to reload config: %v", event.Err)
	case app.StateStopping:
		if event.Interrupted {
			// Move past the "^C" the terminal echoed.
			_, _ = r.output.Write([]byte("\n"))
		}
	case app.StateStopped:
	}
}

// banner is the startup splash: the logo, the development-only disclaimer and
// the mappings the server came up with.
func (r *Renderer) banner(event app.LifecycleEvent) {
	tui.PrintLogo(r.output, r.version)
	r.output.Print("")
	r.output.WarnBox(tui.DisclaimerMessage)
	r.output.Print("")
	r.output.InfoBox(event.Mappings.String())
	r.output.Print("")
}

func (r *Renderer) log(event app.LogEvent) {
	output := r.output
	if event.Prefix != "" {
		output = output.NewPrefixOutput(event.Prefix)
	}

	switch event.Level {
	case app.LevelInfo:
		output.Info(event.Message)
	case app.LevelWarn:
		output.Warn(event.Message)
	case app.LevelError:
		output.Error(event.Message)
	}
}
