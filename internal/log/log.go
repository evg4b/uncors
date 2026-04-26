package log

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

// Level is a logging level.
type Level int32

const (
	// DebugLevel is the debug level.
	DebugLevel Level = -4
	// InfoLevel is the info level.
	InfoLevel Level = 0
	// WarnLevel is the warn level.
	WarnLevel Level = 4
	// ErrorLevel is the error level.
	ErrorLevel Level = 8
	// noLevel is used with log.Print.
	noLevel Level = math.MaxInt32
)

var boxLength = 8

var levelStyles = map[Level]lipgloss.Style{
	DebugLevel: styles.DebugBlockStyle.Width(boxLength),
	InfoLevel:  styles.InfoBlockStyle.Width(boxLength),
	WarnLevel:  styles.WarningBlockStyle.Width(boxLength),
	ErrorLevel: styles.ErrorBlockStyle.Width(boxLength),
	noLevel:    lipgloss.NewStyle(),
}

var messageMap = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

// implements internal.contracts.Logger
type Logger struct {
	w      io.Writer
	level  int32
	prefix string
	mu     *sync.RWMutex
	buf    bytes.Buffer
}

func New(w io.Writer) *Logger {
	return &Logger{
		w:     w,
		level: int32(DebugLevel),
		mu:    &sync.RWMutex{},
		buf:   bytes.Buffer{},
	}
}

func (l *Logger) Error(msg any, keyvals ...any) {
	l.log(ErrorLevel, msg, keyvals...)
}

func (l *Logger) Errorf(msg string, keyvals ...any) {
	l.log(ErrorLevel, fmt.Sprintf(msg, keyvals...))
}

func (l *Logger) Warn(msg any, keyvals ...any) {
	l.log(WarnLevel, msg, keyvals...)
}

func (l *Logger) Warnf(msg string, a ...any) {
	l.log(WarnLevel, fmt.Sprintf(msg, a...))
}

func (l *Logger) Info(msg any, keyvals ...any) {
	l.log(InfoLevel, msg, keyvals...)
}

func (l *Logger) Infof(msg string, a ...any) {
	l.log(InfoLevel, fmt.Sprintf(msg, a...))
}

func (l *Logger) Debug(msg any, keyvals ...any) {
	l.log(DebugLevel, msg, keyvals...)
}

func (l *Logger) Debugf(msg string, a ...any) {
	l.log(DebugLevel, fmt.Sprintf(msg, a...))
}

func (l *Logger) Print(msg any, keyvals ...any) {
	l.log(InfoLevel, msg, keyvals...)
}

func (l *Logger) Printf(msg string, keyvals ...any) {
	l.log(InfoLevel, fmt.Sprintf(msg, keyvals...))
}

func (l *Logger) log(level Level, msg any, keyvals ...any) {
	if atomic.LoadInt32(&l.level) > int32(level) {
		return
	}

	if len(l.prefix) > 0 {
		l.buf.WriteString(l.prefix)
	}

	renderer := levelStyles[level]
	levelMessage := messageMap[level]
	l.buf.WriteString(renderer.Render(levelMessage))

	if strings.HasSuffix(msg.(string), "\n") {
		l.buf.WriteString(fmt.Sprint(msg))
	} else {
		l.buf.WriteString(fmt.Sprintln(msg))
	}

	if err := l.flushBuffer(); err != nil {
		panic(err)
	}
}

func (l *Logger) flushBuffer() error {
	if l.buf.Len() == 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.w.Write(l.buf.Bytes())
	l.buf.Reset()
	return err
}

func (l *Logger) SetLevel(level Level) {
	atomic.StoreInt32(&l.level, int32(level))
}
