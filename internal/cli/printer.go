package cli

import (
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
)

// printerBuffer is how many events the headless printer can fall behind by
// before events are dropped.
const printerBuffer = 1000

// printRequests prints every completed request until the subscription closes.
func printRequests(subscription *events.Subscription, output *tui.CliOutput) {
	for event := range subscription.Events() {
		if completed, ok := event.(events.RequestCompleted); ok {
			output.Request(completed)
		}
	}
}
