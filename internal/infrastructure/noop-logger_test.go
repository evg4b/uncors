package infrastructure_test

import (
	"bytes"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNoopLogger(t *testing.T) {
	testMessage := "test message"
	noopLogger := infrastructure.NoopLogger{}

	t.Run("Infof do nothing", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		noopLogger.Infof(testMessage)

		assert.Empty(t, output.String())
	}))

	t.Run("Errorf do nothing", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		noopLogger.Errorf(testMessage)

		assert.Empty(t, output.String())
	}))
}
