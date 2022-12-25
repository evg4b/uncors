package errordef

import (
	"runtime"
	"strings"
)

type CriticalError struct {
	innerErr   error
	stackTrace string
}

const (
	bufferSize                   = 1 << 16
	criticalErrorTitle           = "Critical error:"
	criticalErrorStackTraceTitle = "Stack trace:"
)

func NewCriticalError(err error) *CriticalError {
	if debug {
		return &CriticalError{
			innerErr:   err,
			stackTrace: collectStackTrace(),
		}
	}

	return &CriticalError{
		innerErr: err,
	}
}

func (err *CriticalError) Error() string {
	builder := &strings.Builder{}
	printLnSafe(builder, criticalErrorTitle, err.innerErr)
	if debug {
		printLnSafe(builder, criticalErrorStackTraceTitle)
		printLnSafe(builder, err.stackTrace)
	}

	return builder.String()
}

func (err *CriticalError) Unwrap() error {
	return err.innerErr
}

func collectStackTrace() string {
	buf := make([]byte, bufferSize)
	runtime.Stack(buf, true)

	return string(buf)
}
