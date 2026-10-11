package telemetry

import "sync"

const (
	SignalSpans = "spans"
	SignalLogs  = "logs"
)

var exportErrors exportErrorHook

// OnExportError registers the function told about the first export error of
// each signal; later ones are dropped, so a missing collector produces one
// notice per run rather than one per batch.
func OnExportError(report func(signal string, err error)) {
	exportErrors.set(report)
}

type exportError struct {
	signal string
	err    error
}

type exportErrorHook struct {
	mu       sync.Mutex
	report   func(signal string, err error)
	pending  []exportError
	reported map[string]bool
}

func (h *exportErrorHook) set(report func(signal string, err error)) {
	h.mu.Lock()
	h.report = report
	h.reported = map[string]bool{}
	pending := h.pending
	h.pending = nil
	h.mu.Unlock()

	for _, pendingErr := range pending {
		h.handle(pendingErr.signal, pendingErr.err)
	}
}

func (h *exportErrorHook) handle(signal string, err error) {
	report := h.claim(signal, err)
	if report != nil {
		report(signal, err)
	}
}

// claim returns the function to report the error with, or nil when the
// signal was reported already or no hook is registered yet.
func (h *exportErrorHook) claim(signal string, err error) func(string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.report == nil {
		// Kept until a hook is registered: Setup runs before the supervisor.
		for _, pendingErr := range h.pending {
			if pendingErr.signal == signal {
				return nil
			}
		}

		h.pending = append(h.pending, exportError{signal: signal, err: err})

		return nil
	}

	if h.reported[signal] {
		return nil
	}

	h.reported[signal] = true

	return h.report
}
