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

func (ServerStarted) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (ServerRestarted) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (ServerStopped) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (ReloadFailed) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (NewVersionAvailable) isEvent() {
	// Seals Event: only types in this package implement it.
}

func (Notice) isEvent() {
	// Seals Event: only types in this package implement it.
}

type Publisher interface {
	Publish(event Event)
}

type NoopPublisher struct{}

func (NoopPublisher) Publish(Event) {
	// Discards the event.
}
