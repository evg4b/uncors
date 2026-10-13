package telemetry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errFirst  = errors.New("first")
	errSecond = errors.New("second")
)

type reported struct {
	signal string
	err    error
}

func TestExportErrorHook(t *testing.T) {
	t.Run("reports the first error of each signal", func(t *testing.T) {
		var (
			hook exportErrorHook
			got  []reported
		)

		hook.set(func(signal string, err error) { got = append(got, reported{signal, err}) })
		hook.handle(SignalLogs, errFirst)
		hook.handle(SignalLogs, errSecond)
		hook.handle(SignalSpans, errSecond)
		hook.handle(SignalSpans, errFirst)

		assert.Equal(t, []reported{{SignalLogs, errFirst}, {SignalSpans, errSecond}}, got)
	})

	t.Run("keeps errors raised before the hook is set", func(t *testing.T) {
		var (
			hook exportErrorHook
			got  []reported
		)

		hook.handle(SignalSpans, errFirst)
		hook.handle(SignalSpans, errSecond)
		hook.set(func(signal string, err error) { got = append(got, reported{signal, err}) })
		hook.handle(SignalSpans, errSecond)

		assert.Equal(t, []reported{{SignalSpans, errFirst}}, got)
	})
}

func TestConfigured(t *testing.T) {
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE", "OTEL_EXPORTER_OTLP_TRACES_INSECURE", "OTEL_EXPORTER_OTLP_LOGS_INSECURE",
	} {
		t.Setenv(key, "")
	}

	assert.False(t, configured("TRACES"))

	t.Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "https://collector:4318/v1/logs")
	assert.False(t, configured("TRACES"))
	assert.True(t, configured("LOGS"))

	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
	assert.True(t, configured("TRACES"))
}
