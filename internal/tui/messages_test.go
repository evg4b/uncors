package tui_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/stretchr/testify/assert"
)

func TestDisclaimerMessage(t *testing.T) {
	assert.NotEmpty(t, tui.DisclaimerMessage)
	assert.Contains(t, tui.DisclaimerMessage, "DON'T USE IT FOR PRODUCTION")
	assert.Contains(t, tui.DisclaimerMessage, "reverse proxy")
	assert.Contains(t, tui.DisclaimerMessage, "security")
}

func TestNewVersionIsAvailable(t *testing.T) {
	assert.NotEmpty(t, tui.NewVersionIsAvailable)
	assert.Contains(t, tui.NewVersionIsAvailable, "NEW VERSION IS AVAILABLE")
	assert.Contains(t, tui.NewVersionIsAvailable, "%s")
}
