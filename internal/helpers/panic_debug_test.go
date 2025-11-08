//go:build !release

package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/stretchr/testify/assert"
)

func TestPanicInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		panicData      any
		shouldBeCalled bool
	}{
		{
			name:           "intercepts panic and return with exit code 3",
			panicData:      "test panic",
			shouldBeCalled: true,
		},
		{
			name:           "intercepts panic and return with exit code 0",
			panicData:      testconstants.ErrTest1,
			shouldBeCalled: true,
		},
	}

	for _, testCast := range tests {
		t.Run(testCast.name, func(t *testing.T) {
			called := false

			assert.Panics(t, func() {
				defer helpers.PanicInterceptor(func(data any) {
					called = true

					assert.Equal(t, testCast.panicData, data)
				})

				panic(testCast.panicData)
			})

			assert.Equal(t, testCast.shouldBeCalled, called)
		})
	}
}
