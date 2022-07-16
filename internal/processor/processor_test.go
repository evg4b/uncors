package processor_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRequestProcessor_HandleRequest(t *testing.T) {
	t.Run("should have correct calling order", func(t *testing.T) {
		tracker := testutils.NewMidelwaresTracker(t)

		requetProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMidelware("middleware1")),
			processor.WithMiddleware(tracker.MakeMidelware("middleware2")),
			processor.WithMiddleware(tracker.MakeFinalMidelware("middleware3")),
		)

		requetProcessor.HandleRequest(nil, nil)

		assert.Equal(t, []string{"middleware1", "middleware2", "middleware3"}, tracker.CallsOrder)
	})

	t.Run("should skip midelwares where next not called", func(t *testing.T) {
		tracker := testutils.NewMidelwaresTracker(t)

		requetProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMidelware("middleware1")),
			processor.WithMiddleware(tracker.MakeFinalMidelware("middleware2")),
			processor.WithMiddleware(tracker.MakeMidelware("middleware3")),
		)

		requetProcessor.HandleRequest(nil, nil)

		assert.Equal(t, []string{"middleware1", "middleware2"}, tracker.CallsOrder)
	})
}
