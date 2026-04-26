package log

import (
	"io"
	"os"
	"sync"
)

var (
	defaultLogger     *Logger
	defaultLoggerOnce sync.Once
)

func Default() *Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = New(os.Stdout)
	})

	return defaultLogger
}

func Null() *Logger {
	return New(io.Discard)
}
