package errordef

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errCriticalErrorTest = errors.New("critical test error")
	errRandom            = errors.New("random")
)

func TestCriticalError(t *testing.T) {
	t.Run("when debug disabled", func(t *testing.T) {
		DisableDebug()
		criticalError := NewCriticalError(errCriticalErrorTest)

		t.Run("should return error", func(t *testing.T) {
			assert.Implements(t, (*error)(nil), criticalError)
		})

		t.Run("should correctly implement unwrapping error", func(t *testing.T) {
			t.Run("unwrap", func(t *testing.T) {
				innerErr := errors.Unwrap(criticalError)

				assert.Equal(t, errCriticalErrorTest, innerErr)
			})

			t.Run("is with same error", func(t *testing.T) {
				actual := errors.Is(criticalError, errCriticalErrorTest)

				assert.True(t, actual)
			})

			t.Run("is with random error", func(t *testing.T) {
				actual := errors.Is(criticalError, errRandom)

				assert.False(t, actual)
			})
		})

		t.Run("should provide error name", func(t *testing.T) {
			actual := criticalError.Error()

			assert.Contains(t, actual, criticalErrorTitle)
		})

		t.Run("should not provide stack trace", func(t *testing.T) {
			actual := criticalError.Error()

			assert.NotContains(t, actual, criticalErrorStackTraceTitle)
		})
	})

	t.Run("when debug enabled", func(t *testing.T) {
		EnableDebug()
		criticalError := NewCriticalError(errCriticalErrorTest)

		t.Run("should provide error name", func(t *testing.T) {
			actual := criticalError.Error()

			assert.Contains(t, actual, criticalErrorTitle)
		})

		t.Run("should provide stack trace", func(t *testing.T) {
			actual := criticalError.Error()

			assert.Contains(t, actual, criticalErrorStackTraceTitle)
		})
	})
}
