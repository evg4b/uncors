package infrastructure_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
)

func TestNoopLogger(t *testing.T) {
	noopLogger := infrastructure.NoopLogger{}

	t.Run("Infof do nothing", func(t *testing.T) {
		noopLogger.Infof("")
	})

	t.Run("Errorf do nothing", func(t *testing.T) {
		noopLogger.Errorf("")
	})
}
