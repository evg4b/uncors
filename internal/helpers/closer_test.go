package helpers_test

import (
	"github.com/evg4b/uncors/testing/testconstants"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCloseSafe(t *testing.T) {
	t.Run("should correct handle nil pointer", func(t *testing.T) {
		assert.NotPanics(t, func() {
			helpers.CloseSafe(nil)
		})
	})

	t.Run("correctly close resource without error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			resource := mocks.NewCloserMock(t).CloseMock.Return(nil)

			helpers.CloseSafe(resource)
		})
	})

	t.Run("panics when resource close return error", func(t *testing.T) {
		assert.Panics(t, func() {
			resource := mocks.NewCloserMock(t).CloseMock.Return(testconstants.ErrTest1)

			helpers.CloseSafe(resource)
		})
	})
}
