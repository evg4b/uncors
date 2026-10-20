package render_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/stretchr/testify/assert"
)

func TestDisclaimerMessage(t *testing.T) {
	assert.NotEmpty(t, render.DisclaimerMessage)
	assert.Contains(t, render.DisclaimerMessage, "DON'T USE IT FOR PRODUCTION")
	assert.Contains(t, render.DisclaimerMessage, "reverse proxy")
	assert.Contains(t, render.DisclaimerMessage, "security")
}

func TestNewVersionIsAvailable(t *testing.T) {
	assert.NotEmpty(t, render.NewVersionIsAvailable)
	assert.Contains(t, render.NewVersionIsAvailable, "NEW VERSION IS AVAILABLE")
	assert.Contains(t, render.NewVersionIsAvailable, "%s")
}
