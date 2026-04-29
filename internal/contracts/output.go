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

type Output interface {
	io.Writer
	InfoOutput
	ErrorOutput
	WarnOutput

	Print(msg any)
	Printf(msg string, args ...any)

	Request(data *ReqestData)

	NewPrefixOutput(prefix string) Output
}
