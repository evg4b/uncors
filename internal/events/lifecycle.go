package events

import "fmt"

// ServerStarted is published once the first configuration is being served.
// Mappings describes what is served; it is a config.Mappings, typed as a
// Stringer because the config package depends on the server.
type ServerStarted struct {
	Mappings fmt.Stringer
}

// ServerRestarted is published once a reloaded configuration is being served.
type ServerRestarted struct {
	Mappings fmt.Stringer
}

// ServerStopped is published once the proxy has stopped serving. It is the
// last event before the bus closes. A proxy that failed to start publishes it
// too; the failure itself is returned to the caller, not published.
type ServerStopped struct{}

// ReloadFailed is published when a configuration change could not be applied.
// The previous configuration stays active.
type ReloadFailed struct {
	Err error
}

// NewVersionAvailable is published when a newer release than the running one
// exists.
type NewVersionAvailable struct {
	Current string
	Latest  string
}

// NoticeLevel is the severity of a Notice.
type NoticeLevel uint8

const (
	NoticeInfo NoticeLevel = iota
	NoticeWarning
	NoticeError
)

// Notice is a message for the user that no other event describes.
type Notice struct {
	Level NoticeLevel
	Text  string
}

func (ServerStarted) isEvent()       {}
func (ServerRestarted) isEvent()     {}
func (ServerStopped) isEvent()       {}
func (ReloadFailed) isEvent()        {}
func (NewVersionAvailable) isEvent() {}
func (Notice) isEvent()              {}

// Publisher receives events. Implementations must not block.
type Publisher interface {
	Publish(event Event)
}

// NoopPublisher discards every event.
type NoopPublisher struct{}

func (NoopPublisher) Publish(Event) {}
