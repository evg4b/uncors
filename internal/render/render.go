// Package render turns application events into console output.
//
// It is the only place that decides how a service event looks. Both run modes
// use it, so neither can drift from the other in what it reports; the service
// itself renders nothing.
package render

import (
	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
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
	case app.RequestEvent:
		r.request(typed)
	}
}

// request renders a completed request. Start events carry no result yet, so
// only the terminal one produces a line - the same rule the console printer
// has always applied.
func (r *Renderer) request(event app.RequestEvent) {
	if !event.Event.Done || event.Event.Data == nil {
		return
	}

	r.withPrefix(event.Event.Prefix).Request(event.Event.Data)
}

func (r *Renderer) lifecycle(event app.LifecycleEvent) {
	switch event.State {
	case app.StateStarting:
		r.banner(event.Mappings)
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
func (r *Renderer) banner(mappings config.Mappings) {
	tui.PrintLogo(r.output, r.version)
	r.output.Print("")
	r.output.WarnBox(tui.DisclaimerMessage)
	r.output.Print("")
	r.output.InfoBox(mappings.String())
	r.output.Print("")
}

// withPrefix returns the output a message carrying prefix is written through.
// An empty prefix renders nothing, so the common case reuses the output rather
// than building one per line.
func (r *Renderer) withPrefix(prefix string) contracts.Output {
	if prefix == "" {
		return r.output
	}

	return r.output.NewPrefixOutput(prefix)
}

func (r *Renderer) log(event app.LogEvent) {
	output := r.withPrefix(event.Prefix)

	switch event.Level {
	case app.LevelInfo:
		output.Info(event.Message)
	case app.LevelWarn:
		output.Warn(event.Message)
	case app.LevelError:
		output.Error(event.Message)
	}
}
