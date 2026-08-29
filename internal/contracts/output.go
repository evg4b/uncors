package contracts

import "io"

type InfoOutput interface {
	Info(msg any)
	Infof(msg string, args ...any)
	InfoBox(messages ...string)
}

type ErrorOutput interface {
	Error(msg any)
	Errorf(msg string, args ...any)
	ErrorBox(messages ...string)
}

type WarnOutput interface {
	Warn(msg any)
	Warnf(msg string, args ...any)
	WarnBox(messages ...string)
}

// PrintOutput writes plain lines.
type PrintOutput interface {
	Print(msg any)
	Printf(msg string, args ...any)
}

// RequestOutput renders one entry of the request activity log.
type RequestOutput interface {
	Request(data *RequestData)
}

// Output is the full console surface. Consumers should accept the narrowest of
// the interfaces above that covers what they actually use; this one exists for
// the composition root and for the run modes, which use all of it.
type Output interface {
	io.Writer
	InfoOutput
	ErrorOutput
	WarnOutput
	PrintOutput
	RequestOutput

	NewPrefixOutput(prefix string) Output
}
