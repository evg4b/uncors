package testutils

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/infra"
)

func LogTest(action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	buffer := &bytes.Buffer{}

	infra.ConfigureLogger()

	logger := log.Default()
	logger.SetOutput(buffer)

	return WithTrueColor(func(t *testing.T) {
		action(t, buffer)
	})
}

func UniqOutput(output *bytes.Buffer, action func(t *testing.T, output *bytes.Buffer)) func(t *testing.T) {
	return func(t *testing.T) {
		t.Cleanup(output.Reset)
		action(t, output)
	}
}
