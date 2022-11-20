//go:build release

package infrastructure

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		panicData      interface{}
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			assert.NotPanics(t, func() {
				defer PanicInterceptor(func(data interface{}) {
					called = true
					assert.Equal(t, tt.panicData, data)
				})

				panic(tt.panicData)
			})

			assert.Equal(t, tt.shouldBeCalled, called)
		})
	}
}
