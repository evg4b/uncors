package log

import (
	"os"
	"sync"
)

var (
	defaultLogger     *Logger = nil
	defaultLoggerOnce sync.Once
)

func Default() *Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = New(os.Stdout)
	})
	return defaultLogger
}
