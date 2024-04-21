package uncors_test

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestLoggers(t *testing.T) {
	t.Run("NewProxyLogger", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		log.SetLevel(log.DebugLevel)

		tests := []struct {
			name     string
			logger   *log.Logger
			expected string
		}{
			{
				name:     "ProxyLogger",
				logger:   uncors.NewProxyLogger(log.Default()),
				expected: styles.ProxyStyle.Render("PROXY"),
			},
			{
				name:     "MockLogger",
				logger:   uncors.NewMockLogger(log.Default()),
				expected: styles.MockStyle.Render("MOCK"),
			},
			{
				name:     "StaticLogger",
				logger:   uncors.NewStaticLogger(log.Default()),
				expected: styles.StaticStyle.Render("STATIC"),
			},
			{
				name:     "CacheLogger",
				logger:   uncors.NewCacheLogger(log.Default()),
				expected: styles.CacheStyle.Render("CACHE"),
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				t.Run("Error", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Error("error")

					assert.Contains(t, output.String(), testCase.expected)
				}))

				t.Run("Warn", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Warn("warn")

					assert.Contains(t, output.String(), testCase.expected)
				}))

				t.Run("Info", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Info("info")

					assert.Contains(t, output.String(), testCase.expected)
				}))

				t.Run("Debug", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Debug("debug")

					assert.Contains(t, output.String(), testCase.expected)
				}))

				t.Run("Print", testutils.UniqOutput(output, func(t *testing.T, output *bytes.Buffer) {
					testCase.logger.Print("print")

					assert.Contains(t, output.String(), testCase.expected)
				}))
			})
		}
	}))
}
