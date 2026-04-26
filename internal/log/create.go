package log

import (
	"bytes"
	"sync/atomic"
)

func CreateLogger(logger *Logger, prefix string) *Logger {
	return &Logger{
		w:      logger.w,
		level:  atomic.LoadInt32(&logger.level),
		mu:     logger.mu,
		buf:    bytes.Buffer{},
		prefix: prefix,
	}
}
