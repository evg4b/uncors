package uncors

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/tui"
)

func WithLogPrinter(printer tui.Printer) Option {
	return func(model *uncorsModel) {
		model.logPrinter = printer
	}
}

func WithVersion(version string) Option {
	return func(model *uncorsModel) {
		model.version = version
	}
}

func WithConfig(config *config.UncorsConfig) Option {
	return func(model *uncorsModel) {
		model.config = config
	}
}

func WithRequestTracker(tracker tui.RequestTracker) Option {
	return func(model *uncorsModel) {
		model.requestTracker = tracker
	}
}
