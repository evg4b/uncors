package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeLogValue(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{name: "plain value is unchanged", in: "http://example.com/path", expected: "http://example.com/path"},
		{name: "line feed is replaced", in: "a\nb", expected: "a b"},
		{name: "carriage return is replaced", in: "a\rb", expected: "a b"},
		{name: "forged log entry is neutralised", in: "/x\r\nERROR: injected", expected: "/x  ERROR: injected"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, helpers.SanitizeLogValue(testCase.in))
		})
	}
}
