package helpers_test

import (
	"errors"
	"testing"

	"github.com/evg4b/uncors/internal/errordef"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

var errTest = errors.New("test error")

var doAction = func() {
	helpers.HandleCriticalError(errTest)
}

func TestHandleCriticalError(t *testing.T) {
	t.Run("where error is nil", func(t *testing.T) {
		t.Run("should not panic", func(t *testing.T) {
			assert.NotPanics(t, func() {
				helpers.HandleCriticalError(nil)
			})
		})
	})

	t.Run("where error is defined", func(t *testing.T) {
		t.Run("should panic", func(t *testing.T) {
			assert.Panics(t, doAction)
		})
		t.Run("should panic with critical error", func(t *testing.T) {
			testutils.PanicWith(t, doAction, func(err any) {
				assert.IsType(t, &errordef.CriticalError{}, err)
			})
		})
	})
}
