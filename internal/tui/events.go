package tui

import (
	"github.com/evg4b/uncors/internal/events"
)

// EventRenderer prints bus events the way both frontends show them: the
// headless printer straight to the terminal, the interactive UI into its
// history.
type EventRenderer struct {
	output  *CliOutput
	version string
}

func NewEventRenderer(output *CliOutput, version string) *EventRenderer {
	return &EventRenderer{output: output, version: version}
}

// Render prints event. Events with nothing to show, such as RequestStarted,
// print nothing.
func (r *EventRenderer) Render(event events.Event) {
	switch typed := event.(type) {
	case events.RequestCompleted:
		r.output.Request(typed)
	case events.ServerStarted:
		PrintLogo(r.output, r.version)
		r.output.Print("")
		r.output.WarnBox(DisclaimerMessage)
		r.output.Print("")
		r.output.InfoBox(typed.Mappings.String())
		r.output.Print("")
	case events.ServerRestarted:
		r.output.InfoBox("Server restarted", typed.Mappings.String())
	case events.ServerStopped:
		r.output.Info("Server was stopped")
	case events.ReloadFailed:
		r.output.Errorf("Failed to reload configuration, keeping the previous one: %v", typed.Err)
	case events.NewVersionAvailable:
		r.output.Infof(NewVersionIsAvailable, typed.Current, typed.Latest)
		r.output.Print("")
	case events.Notice:
		r.renderNotice(typed)
	}
}

func (r *EventRenderer) renderNotice(notice events.Notice) {
	switch notice.Level {
	case events.NoticeWarning:
		r.output.Warn(notice.Text)
	case events.NoticeError:
		r.output.Error(notice.Text)
	case events.NoticeInfo:
		r.output.Info(notice.Text)
	}
}
