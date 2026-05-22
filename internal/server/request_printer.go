package server

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func RequestPrinter(tracker *RequestTracker, output contracts.Output) {
	for event := range tracker.Events() {
		if event.Done && event.Data != nil {
			if event.Prefix != "" {
				output.NewPrefixOutput(event.Prefix).Request(event.Data)
			} else {
				output.Request(event.Data)
			}
		}
	}
}
