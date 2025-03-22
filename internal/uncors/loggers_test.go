package uncors_test

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"
)

func TestLoggers(t *testing.T) {
	t.Run("NewProxyLogger", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		log.SetLevel(log.DebugLevel)

		tests := []struct {
			name   string
			logger *log.Logger
		}{
			{
				name:   "ProxyLogger",
				logger: uncors.NewProxyLogger(log.Default()),
			},
			{
				name:   "MockLogger",
				logger: uncors.NewMockLogger(log.Default()),
			},
			{
				name:   "StaticLogger",
				logger: uncors.NewStaticLogger(log.Default()),
			},
			{
				name:   "CacheLogger",
				logger: uncors.NewCacheLogger(log.Default()),
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				t.Run("Error", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Error("error")

					snaps.MatchSnapshot(t, output.String())
				}))

				t.Run("Warn", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Warn("warn")

					snaps.MatchSnapshot(t, output.String())
				}))

				t.Run("Info", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Info("info")

					snaps.MatchSnapshot(t, output.String())
				}))

				t.Run("Debug", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Debug("debug")

					snaps.MatchSnapshot(t, output.String())
				}))

				t.Run("Print", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Print("print")

					snaps.MatchSnapshot(t, output.String())
				}))
			})
		}
	}))
}
