package tui_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/charmbracelet/log"

	"github.com/evg4b/uncors/internal/contracts"
	new_log "github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/muesli/termenv"
)

func newTestLogger(buf *bytes.Buffer) *log.Logger {
	logger := log.New(buf)
	logger.SetReportTimestamp(false)
	logger.SetReportCaller(false)
	logger.SetStyles(&tui.DefaultStyles)
	logger.SetLevel(log.DebugLevel)
	logger.SetColorProfile(termenv.TrueColor)
	return logger
}

// TestDefaultStyles captures the rendered output of DefaultStyles for every log level.
// These snapshots serve as the specification for the new logger implementation.
func TestDefaultStyles(t *testing.T) {
	tests := []struct {
		name string
		fn   func(contracts.Logger)
	}{
		{"debug", func(l contracts.Logger) { l.Debug("test message") }},
		{"info", func(l contracts.Logger) { l.Info("test message") }},
		{"warn", func(l contracts.Logger) { l.Warn("test message") }},
		{"error", func(l contracts.Logger) { l.Error("test message") }},
		//{"fatal", func(l contracts.Logger) { l.Log(log.FatalLevel, "test message") }},
		{"print (no level)", func(l contracts.Logger) { l.Print("test message") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := new_log.New(buf)
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

// TestDefaultStylesWithKeyValues captures output when structured key-value pairs are logged.
func TestDefaultStylesWithKeyValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debug with kv", func(l *log.Logger) { l.Debug("test message", "key", "value") }},
		{"info with kv", func(l *log.Logger) { l.Info("test message", "key", "value") }},
		{"warn with kv", func(l *log.Logger) { l.Warn("test message", "key", "value") }},
		{"error with kv", func(l *log.Logger) { l.Error("test message", "key", "value") }},
		{"print with kv (no level)", func(l *log.Logger) { l.Print("test message", "key", "value") }},
		{
			"multiple key-values",
			func(l *log.Logger) { l.Info("test message", "key1", "val1", "key2", "val2") },
		},
		{
			"error value",
			func(l *log.Logger) { l.Error("something failed", "err", fmt.Errorf("connection refused")) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := newTestLogger(buf)
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

type prefixDef struct {
	name   string
	render func() string
}

func allPrefixes() []prefixDef {
	return []prefixDef{
		{"no prefix", func() string { return "" }},
		{"proxy", func() string { return styles.ProxyStyle.Render("PROXY") }},
		{"mock", func() string { return styles.MockStyle.Render("MOCK") }},
		{"static", func() string { return styles.StaticStyle.Render("STATIC") }},
		{"cache", func() string { return styles.CacheStyle.Render("CACHE") }},
		{"rewrite", func() string { return styles.RewriteStyle.Render("REWRITE") }},
	}
}

// TestCreateLogger captures logger output for every combination of prefix style × log level.
// Prefix rendering happens inside WithTrueColor so that 24-bit color codes are present.
func TestCreateLogger(t *testing.T) {
	levels := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debug", func(l *log.Logger) { l.Debug("test message") }},
		{"info", func(l *log.Logger) { l.Info("test message") }},
		{"warn", func(l *log.Logger) { l.Warn("test message") }},
		{"error", func(l *log.Logger) { l.Error("test message") }},
		{"fatal", func(l *log.Logger) { l.Log(log.FatalLevel, "test message") }},
		{"print (no level)", func(l *log.Logger) { l.Print("test message") }},
	}

	for _, p := range allPrefixes() {
		t.Run(p.name, testutils.WithTrueColor(func(t *testing.T) {
			prefix := p.render() // rendered with TrueColor active
			for _, lvl := range levels {
				t.Run(lvl.name, func(t *testing.T) {
					buf := &bytes.Buffer{}
					base := newTestLogger(buf)
					logger := tui.CreateLogger(base, prefix)
					lvl.fn(logger)
					testutils.MatchSnapshot(t, buf.String())
				})
			}
		}))
	}
}

// TestCreateLoggerWithKeyValues captures logger output for every combination of prefix × structured kv logging.
func TestCreateLoggerWithKeyValues(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{
			"single key-value",
			func(l *log.Logger) { l.Info("test message", "key", "value") },
		},
		{
			"multiple key-values",
			func(l *log.Logger) { l.Info("test message", "key1", "val1", "key2", "val2") },
		},
		{
			"debug with key-value",
			func(l *log.Logger) { l.Debug("debug details", "component", "router") },
		},
		{
			"warn with key-value",
			func(l *log.Logger) { l.Warn("slow response", "latency_ms", 1500) },
		},
		{
			"error with error value",
			func(l *log.Logger) { l.Error("request failed", "err", fmt.Errorf("connection refused")) },
		},
		{
			"print with key-value (no level)",
			func(l *log.Logger) { l.Print("proxying", "from", "localhost:8080", "to", "api.example.com") },
		},
	}

	for _, p := range allPrefixes() {
		t.Run(p.name, testutils.WithTrueColor(func(t *testing.T) {
			prefix := p.render()
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					buf := &bytes.Buffer{}
					base := newTestLogger(buf)
					logger := tui.CreateLogger(base, prefix)
					c.fn(logger)
					testutils.MatchSnapshot(t, buf.String())
				})
			}
		}))
	}
}
