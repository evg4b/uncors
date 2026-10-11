package events

import "fmt"

type ServerStarted struct {
	// config.Mappings, typed as a Stringer because config depends on server.
	Mappings fmt.Stringer
}

type ServerRestarted struct {
	Mappings fmt.Stringer
}

type ServerStopped struct{}

type ReloadFailed struct {
	Err error
}

type NewVersionAvailable struct {
	Current string
	Latest  string
}

type NoticeLevel uint8

const (
	NoticeInfo NoticeLevel = iota
	NoticeWarning
	NoticeError
)

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

type Publisher interface {
	Publish(event Event)
}

type NoopPublisher struct{}

func (NoopPublisher) Publish(Event) {}
