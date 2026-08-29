package infra

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o600
)

// LogOptions describe where diagnostics go and how much of them is kept.
type LogOptions struct {
	// Level is one of debug, info, warn or error.
	Level string
	// File, when set, receives the diagnostics instead of stderr.
	File string
	// Handler overrides both of the above; the terminal UI uses it to route
	// records into its own view instead of writing over the alt-screen.
	Handler slog.Handler
}

// SetupLogging installs the process wide logger and returns the log file, if one
// was opened, for the caller to close.
//
// Diagnostics go to stderr by default. Discarding them by default is what made a
// dead config watcher, a HAR archive that never got written and a rejected TLS
// handshake all look exactly like success.
func SetupLogging(options LogOptions) (io.Closer, error) {
	level, err := ParseLogLevel(options.Level)
	if err != nil {
		return nil, err
	}

	if options.Handler != nil {
		installLogger(slog.New(options.Handler))

		return nil, nil //nolint:nilnil // the caller owns no file in this case
	}

	var (
		writer io.Writer = os.Stderr
		closer io.Closer
	)

	if options.File != "" {
		file, openErr := os.OpenFile(filepath.Clean(options.File), logFileFlags, logFilePerm)
		if openErr != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %w", options.File, openErr)
		}

		writer, closer = file, file
	}

	installLogger(slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})))

	return closer, nil
}

// ParseLogLevel maps a level name to its slog level.
func ParseLogLevel(name string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w: %q (want debug, info, warn or error)", ErrUnknownLogLevel, name)
	}
}

// StdLogger adapts the process logger to the *log.Logger that net/http wants for
// http.Server.ErrorLog, so that TLS handshake failures and recovered handler
// panics stop being invisible.
func StdLogger() *log.Logger {
	return slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn)
}

func installLogger(logger *slog.Logger) {
	// SetDefault also routes the standard log package through this handler, so
	// any log.Printf left in the tree still reaches the same place.
	slog.SetDefault(logger)
}
