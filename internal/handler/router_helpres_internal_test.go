package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceWildcards(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{name: "empty string", host: "", expected: ""},
		{name: "host without wildcard", host: "demo.com", expected: "demo.com"},
		{name: "host with wildcard", host: "*.demo.com", expected: "{p1}.demo.com"},
		{name: "host with multiple wildcard", host: "*.*.demo*.*", expected: "{p1}.{p2}.demo{p3}.{p4}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := replaceWildcards(tt.host)

			assert.Equal(t, tt.expected, actual)
		})
	}
}
