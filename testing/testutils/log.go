package testutils

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
)

func LogTest(action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	buffer := &bytes.Buffer{}
	log.SetOutput(buffer)

	return func(t *testing.T) {
		action(t, buffer)
	}
}
