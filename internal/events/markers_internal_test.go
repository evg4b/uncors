package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkers(t *testing.T) {
	allEvents := []Event{
		RequestStarted{},
		RequestCompleted{},
		ServerStarted{},
		ServerRestarted{},
		ServerStopped{},
		ReloadFailed{},
		NewVersionAvailable{},
		Notice{},
	}

	for _, event := range allEvents {
		assert.NotPanics(t, event.isEvent)
	}

	for _, command := range []Command{Restart{}, Shutdown{}} {
		assert.NotPanics(t, command.isCommand)
	}
}

func TestNoopPublisher(t *testing.T) {
	assert.NotPanics(t, func() {
		NoopPublisher{}.Publish(ServerStopped{})
	})
}
