//go:build release

package infra_test

import (
	"errors"
	"github.com/evg4b/uncors/internal/infra"
	"testing"

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
			panicData:      errors.New("test error"),
			shouldBeCalled: true,
		},
		{
			name:      "intercepts panic and return with exit code 0",
			panicData: nil,
		},
	}

	for _, testCast := range tests {
		t.Run(testCast.name, func(t *testing.T) {
			called := false

			assert.NotPanics(t, func() {
				defer infra.PanicInterceptor(func(data any) {
					called = true
					assert.Equal(t, testCast.panicData, data)
				})

				panic(testCast.panicData)
			})

			assert.Equal(t, testCast.shouldBeCalled, called)
		})
	}
}
