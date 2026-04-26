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
	"github.com/evg4b/uncors/internal/tui"
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
	DebugLevel: styles.DebugBlockStyle.Width(boxLength).Bold(true),
	InfoLevel:  styles.InfoBlockStyle.Width(boxLength).Bold(true),
	WarnLevel:  styles.WarningBlockStyle.Width(boxLength).Bold(true),
	ErrorLevel: styles.ErrorBlockStyle.Width(boxLength).Bold(true),
	noLevel:    lipgloss.NewStyle(),
}

var faintStyle = lipgloss.NewStyle().Faint(true)

var messageMap = map[Level]string{
	DebugLevel: tui.DebugLabel,
	InfoLevel:  tui.InfoLabel,
	WarnLevel:  tui.WarningLabel,
	ErrorLevel: tui.ErrorLabel,
}

// Logger implements contracts.Logger.
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
	l.log(ErrorLevel, fmt.Sprint(msg), keyvals...)
}

func (l *Logger) Errorf(msg string, args ...any) {
	l.log(ErrorLevel, fmt.Sprintf(msg, args...))
}

func (l *Logger) Warn(msg any, keyvals ...any) {
	l.log(WarnLevel, fmt.Sprint(msg), keyvals...)
}

func (l *Logger) Warnf(msg string, args ...any) {
	l.log(WarnLevel, fmt.Sprintf(msg, args...))
}

func (l *Logger) Info(msg any, keyvals ...any) {
	l.log(InfoLevel, fmt.Sprint(msg), keyvals...)
}

func (l *Logger) Infof(msg string, args ...any) {
	l.log(InfoLevel, fmt.Sprintf(msg, args...))
}

func (l *Logger) Debug(msg any, keyvals ...any) {
	l.log(DebugLevel, fmt.Sprint(msg), keyvals...)
}

func (l *Logger) Debugf(msg string, args ...any) {
	l.log(DebugLevel, fmt.Sprintf(msg, args...))
}

func (l *Logger) Print(msg any, keyvals ...any) {
	l.log(noLevel, fmt.Sprint(msg), keyvals...)
}

func (l *Logger) Printf(msg string, args ...any) {
	l.log(noLevel, fmt.Sprintf(msg, args...))
}

func (l *Logger) SetLevel(level Level) {
	atomic.StoreInt32(&l.level, int32(level))
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.w = w
}

func (l *Logger) log(level Level, msg string, keyvals ...any) {
	if atomic.LoadInt32(&l.level) > int32(level) {
		return
	}

	l.renderPrefix()
	l.renderLevel(level)
	l.renderMessage(msg)
	l.renderKeyVals(keyvals)
	l.buf.WriteByte('\n')

	err := l.flushBuffer()
	if err != nil {
		panic(err)
	}
}

func (l *Logger) renderKeyVals(keyvals []any) {
	if len(keyvals) < 1 {
		return
	}

	for kvIdx := 0; kvIdx < len(keyvals); kvIdx += 2 {
		l.buf.WriteByte(' ')

		if kvIdx+1 >= len(keyvals) {
			context := fmt.Sprintf("%v=undefined-value", keyvals[kvIdx])
			l.buf.WriteString(faintStyle.Render(context))
		} else {
			context := fmt.Sprintf("%v=%+v", keyvals[kvIdx], keyvals[kvIdx+1])
			l.buf.WriteString(faintStyle.Render(context))
		}
	}
}

func (l *Logger) renderMessage(msg string) {
	msg = strings.TrimSuffix(msg, "\n")
	fmt.Fprint(&l.buf, msg)
}

func (l *Logger) renderLevel(level Level) {
	renderer := levelStyles[level]
	levelMessage := messageMap[level]
	l.buf.WriteString(renderer.Render(levelMessage))
	l.buf.WriteByte(' ')
}

func (l *Logger) renderPrefix() {
	if len(l.prefix) > 0 {
		l.buf.WriteString(l.prefix)
	}
}

func (l *Logger) flushBuffer() error {
	if l.buf.Len() == 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	defer l.buf.Reset()

	_, err := l.w.Write(l.buf.Bytes())

	return err
}
