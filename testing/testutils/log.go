package testutils

import (
	"bytes"
	"github.com/evg4b/uncors/internal/log"
	"testing"
)

func LogTest(action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	var buffer = &bytes.Buffer{}
	log.SetOutput(buffer)

	return func(t *testing.T) {
		action(t, buffer)
	}
}
