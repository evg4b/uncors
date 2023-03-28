package server_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/server"

	"github.com/go-playground/assert/v2"
)

func TestAtomicBool(t *testing.T) {
	tests := []struct {
		name     string
		getBool  func() *server.AtomicBool
		expected bool
	}{
		{
			name: "Should be false by default",
			getBool: func() *server.AtomicBool {
				var value server.AtomicBool

				return &value
			},
			expected: false,
		},
		{
			name: "Should be false after SetFalse",
			getBool: func() *server.AtomicBool {
				var value server.AtomicBool
				value.SetFalse()

				return &value
			},
			expected: false,
		},
		{
			name: "Should be true after SetTrue",
			getBool: func() *server.AtomicBool {
				var value server.AtomicBool
				value.SetTrue()

				return &value
			},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := tt.getBool()

			assert.IsEqual(value.IsSet(), tt.expected)
		})
	}
}
