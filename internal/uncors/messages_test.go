package uncors_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/uncors"
	"github.com/stretchr/testify/assert"
)

func TestDisclaimerMessage(t *testing.T) {
	assert.NotEmpty(t, uncors.DisclaimerMessage)
	assert.Contains(t, uncors.DisclaimerMessage, "DON'T USE IT FOR PRODUCTION")
	assert.Contains(t, uncors.DisclaimerMessage, "reverse proxy")
	assert.Contains(t, uncors.DisclaimerMessage, "security")
}

func TestNewVersionIsAvailable(t *testing.T) {
	assert.NotEmpty(t, uncors.NewVersionIsAvailable)
	assert.Contains(t, uncors.NewVersionIsAvailable, "NEW VERSION IS AVAILABLE")
	assert.Contains(t, uncors.NewVersionIsAvailable, "%s")
}
