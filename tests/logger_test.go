package logger_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/testutils"
)

func newBase() (*bytes.Buffer, *log.Logger) {
	buf := &bytes.Buffer{}
	logger := log.New(buf)
	logger.SetLevel(log.DebugLevel)

	return buf, logger
}

// TestLoggerLevels covers every log level using the plain (non-format) methods.
func TestLoggerLevels(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debug", func(l *log.Logger) { l.Debug("test message") }},
		{"info", func(l *log.Logger) { l.Info("test message") }},
		{"warn", func(l *log.Logger) { l.Warn("test message") }},
		{"error", func(l *log.Logger) { l.Error("test message") }},
		{"print (no level)", func(l *log.Logger) { l.Print("test message") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf, logger := newBase()
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

// TestLoggerFormattedMethods covers every level using the format-string (f-suffix) methods.
func TestLoggerFormattedMethods(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debugf", func(l *log.Logger) { l.Debugf("hello %s, count=%d", "world", 42) }},
		{"infof", func(l *log.Logger) { l.Infof("hello %s, count=%d", "world", 42) }},
		{"warnf", func(l *log.Logger) { l.Warnf("hello %s, count=%d", "world", 42) }},
		{"errorf", func(l *log.Logger) { l.Errorf("hello %s, count=%d", "world", 42) }},
		{"printf (no level)", func(l *log.Logger) { l.Printf("hello %s, count=%d", "world", 42) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf, logger := newBase()
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

// TestLoggerKeyValues covers structured key-value logging across levels,
// including edge cases: multiple pairs, odd count (missing value), error values.
func TestLoggerKeyValues(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debug with kv", func(l *log.Logger) { l.Debug("details", "component", "router") }},
		{"info single kv", func(l *log.Logger) { l.Info("request handled", "status", 200) }},
		{"info two kv pairs", func(l *log.Logger) { l.Info("request handled", "status", 200, "latency_ms", 42) }},
		{"info three kv pairs", func(l *log.Logger) { l.Info("request handled", "status", 200, "latency_ms", 42, "path", "/api") }},
		{"warn with kv", func(l *log.Logger) { l.Warn("slow response", "latency_ms", 1500) }},
		{"error with kv", func(l *log.Logger) { l.Error("request failed", "status", 500) }},
		{"error with error value", func(l *log.Logger) { l.Error("failed", "err", fmt.Errorf("connection refused")) }},
		{"print with kv (no level)", func(l *log.Logger) { l.Print("proxying", "from", "localhost:8080", "to", "api.example.com") }},
		{"odd kv count (missing value)", func(l *log.Logger) { l.Info("unexpected", "orphan-key") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf, logger := newBase()
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

// TestLoggerMessageVariants covers special message content:
// trailing newlines (stripped by the renderer), empty strings, non-string types.
func TestLoggerMessageVariants(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"trailing newline stripped", func(l *log.Logger) { l.Info("message with newline\n") }},
		{"empty message", func(l *log.Logger) { l.Info("") }},
		{"numeric message", func(l *log.Logger) { l.Info(42) }},
		{"error as message", func(l *log.Logger) { l.Error(fmt.Errorf("disk full")) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf, logger := newBase()
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

// TestLoggerLevelFiltering verifies that SetLevel suppresses lower-priority messages
// and passes through messages at or above the configured level.
func TestLoggerLevelFiltering(t *testing.T) {
	cases := []struct {
		name     string
		setLevel log.Level
		fn       func(*log.Logger)
	}{
		{"debug suppressed at info", log.InfoLevel, func(l *log.Logger) { l.Debug("hidden") }},
		{"info suppressed at warn", log.WarnLevel, func(l *log.Logger) { l.Info("hidden") }},
		{"warn suppressed at error", log.ErrorLevel, func(l *log.Logger) { l.Warn("hidden") }},
		{"info passes at info", log.InfoLevel, func(l *log.Logger) { l.Info("visible") }},
		{"warn passes at warn", log.WarnLevel, func(l *log.Logger) { l.Warn("visible") }},
		{"error passes at error", log.ErrorLevel, func(l *log.Logger) { l.Error("visible") }},
		{"error passes above warn", log.WarnLevel, func(l *log.Logger) { l.Error("visible") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, testutils.WithTrueColor(func(t *testing.T) {
			buf, logger := newBase()
			logger.SetLevel(tc.setLevel)
			tc.fn(logger)
			testutils.MatchSnapshot(t, buf.String())
		}))
	}
}

type loggerFactory struct {
	name   string
	create func(*log.Logger) *log.Logger
}

func allLoggerFactories() []loggerFactory {
	return []loggerFactory{
		{"no prefix", func(base *log.Logger) *log.Logger { return base }},
		{"proxy", uncors.NewProxyLogger},
		{"options", uncors.NewOptionsLogger},
		{"mock", uncors.NewMockLogger},
		{"static", uncors.NewStaticLogger},
		{"cache", uncors.NewCacheLogger},
		{"rewrite", uncors.NewRewriteLogger},
		{"script", uncors.NewScriptLogger},
	}
}

// TestNamedLoggerLevels covers every named logger × log level combination for plain messages.
func TestNamedLoggerLevels(t *testing.T) {
	levels := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debug", func(l *log.Logger) { l.Debug("test message") }},
		{"info", func(l *log.Logger) { l.Info("test message") }},
		{"warn", func(l *log.Logger) { l.Warn("test message") }},
		{"error", func(l *log.Logger) { l.Error("test message") }},
		{"print (no level)", func(l *log.Logger) { l.Print("test message") }},
	}

	for _, f := range allLoggerFactories() {
		t.Run(f.name, testutils.WithTrueColor(func(t *testing.T) {
			for _, lvl := range levels {
				t.Run(lvl.name, func(t *testing.T) {
					buf, base := newBase()
					logger := f.create(base)
					lvl.fn(logger)
					testutils.MatchSnapshot(t, buf.String())
				})
			}
		}))
	}
}

// TestNamedLoggerFormattedMethods covers every named logger × format-string method combination.
func TestNamedLoggerFormattedMethods(t *testing.T) {
	methods := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"debugf", func(l *log.Logger) { l.Debugf("hello %s, count=%d", "world", 42) }},
		{"infof", func(l *log.Logger) { l.Infof("hello %s, count=%d", "world", 42) }},
		{"warnf", func(l *log.Logger) { l.Warnf("hello %s, count=%d", "world", 42) }},
		{"errorf", func(l *log.Logger) { l.Errorf("hello %s, count=%d", "world", 42) }},
		{"printf (no level)", func(l *log.Logger) { l.Printf("hello %s, count=%d", "world", 42) }},
	}

	for _, f := range allLoggerFactories() {
		t.Run(f.name, testutils.WithTrueColor(func(t *testing.T) {
			for _, m := range methods {
				t.Run(m.name, func(t *testing.T) {
					buf, base := newBase()
					logger := f.create(base)
					m.fn(logger)
					testutils.MatchSnapshot(t, buf.String())
				})
			}
		}))
	}
}

// TestNamedLoggerKeyValues covers every named logger × key-value variant combination.
func TestNamedLoggerKeyValues(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*log.Logger)
	}{
		{"info single kv", func(l *log.Logger) { l.Info("request handled", "status", 200) }},
		{"info multiple kv", func(l *log.Logger) { l.Info("request handled", "status", 200, "latency_ms", 42) }},
		{"debug with kv", func(l *log.Logger) { l.Debug("component started", "name", "router") }},
		{"warn with kv", func(l *log.Logger) { l.Warn("slow response", "latency_ms", 1500) }},
		{"error with error value", func(l *log.Logger) { l.Error("request failed", "err", fmt.Errorf("connection refused")) }},
		{"print with kv (no level)", func(l *log.Logger) { l.Print("proxying", "from", "localhost:8080", "to", "api.example.com") }},
		{"odd kv count (missing value)", func(l *log.Logger) { l.Info("unexpected", "orphan-key") }},
	}

	for _, f := range allLoggerFactories() {
		t.Run(f.name, testutils.WithTrueColor(func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					buf, base := newBase()
					logger := f.create(base)
					c.fn(logger)
					testutils.MatchSnapshot(t, buf.String())
				})
			}
		}))
	}
}
