package testutils

import (
	"testing"
)

func WithTrueColor(action func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		action(t)
	}
}
