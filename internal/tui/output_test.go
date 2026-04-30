package tui_test

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errWrite = fmt.Errorf("write error")

type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errWrite
}

func TestCliOutput_Write(t *testing.T) {
	var buf strings.Builder

	out := tui.NewCliOutput(&buf)

	n, err := out.Write([]byte("direct write"))

	require.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, "direct write", buf.String())
}

func TestCliOutput_Info(t *testing.T) {
	t.Run("Info", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Info("test message")
		testutils.MatchSnapshot(t, buf.String())
	}))

	t.Run("Infof", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Infof("formatted %s %d", "message", 42)
		testutils.MatchSnapshot(t, buf.String())
	}))
}

func TestCliOutput_Error(t *testing.T) {
	t.Run("Error", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Error("test error")
		testutils.MatchSnapshot(t, buf.String())
	}))

	t.Run("Errorf", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Errorf("formatted %s %d", "error", 42)
		testutils.MatchSnapshot(t, buf.String())
	}))
}

func TestCliOutput_Warn(t *testing.T) {
	t.Run("Warn", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Warn("test warning")
		testutils.MatchSnapshot(t, buf.String())
	}))

	t.Run("Warnf", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Warnf("formatted %s %d", "warning", 42)
		testutils.MatchSnapshot(t, buf.String())
	}))
}

func TestCliOutput_Print(t *testing.T) {
	t.Run("Print", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Print("plain message")
		testutils.MatchSnapshot(t, buf.String())
	}))

	t.Run("Printf", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf).Printf("formatted %s %d", "plain", 42)
		testutils.MatchSnapshot(t, buf.String())
	}))
}

func TestCliOutput_RenderMessage_StripsTrailingNewline(t *testing.T) {
	var buf strings.Builder
	tui.NewCliOutput(&buf).Info("hello\n")
	output := buf.String()
	assert.Contains(t, output, "hello")
	assert.NotContains(t, output, "hello\n\n")
}

func TestCliOutput_WithPrefix(t *testing.T) {
	t.Run("prefix appears in output", func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf, tui.WithPrefix("[pfx]")).Info("message")
		assert.Contains(t, buf.String(), "[pfx]")
	})

	t.Run("snapshot with prefix", testutils.WithTrueColor(func(t *testing.T) {
		var buf strings.Builder
		tui.NewCliOutput(&buf, tui.WithPrefix("[pfx]")).Info("message")
		testutils.MatchSnapshot(t, buf.String())
	}))

	t.Run("empty prefix is not written", func(t *testing.T) {
		var (
			withPrefix    strings.Builder
			withoutPrefix strings.Builder
		)

		tui.NewCliOutput(&withPrefix, tui.WithPrefix("")).Info("message")
		tui.NewCliOutput(&withoutPrefix).Info("message")
		assert.Equal(t, withoutPrefix.String(), withPrefix.String())
	})
}

func TestCliOutput_NewPrefixOutput(t *testing.T) {
	t.Run("writes to parent writer", func(t *testing.T) {
		var buf strings.Builder

		parent := tui.NewCliOutput(&buf)
		sub := parent.NewPrefixOutput("[sub]")
		parent.Print("from parent")
		sub.Print("from sub")

		output := buf.String()
		assert.Contains(t, output, "from parent")
		assert.Contains(t, output, "from sub")
	})

	t.Run("sub prefix appears in output", func(t *testing.T) {
		var buf strings.Builder

		parent := tui.NewCliOutput(&buf)
		parent.NewPrefixOutput("[sub]").Print("msg")
		assert.Contains(t, buf.String(), "[sub]")
	})

	t.Run("implements Output interface", func(_ *testing.T) {
		parent := tui.NewCliOutput(io.Discard)

		_ = parent.NewPrefixOutput("[sub]")
	})
}

func TestCliOutput_PanicsOnWriteError(t *testing.T) {
	out := tui.NewCliOutput(&errorWriter{})

	assert.Panics(t, func() {
		out.Info("this will panic")
	})
}

func TestCliOutput_ConcurrentSafety(t *testing.T) {
	var buf strings.Builder

	parent := tui.NewCliOutput(&buf)

	const goroutines = 20

	var waitGroup sync.WaitGroup

	for goroutine := range goroutines {
		waitGroup.Add(1)

		go func(idx int) {
			defer waitGroup.Done()

			out := parent.NewPrefixOutput(fmt.Sprintf("[g%d]", idx))
			out.Info(fmt.Sprintf("message from goroutine %d", idx))
		}(goroutine)
	}

	waitGroup.Wait()
	assert.NotEmpty(t, buf.String())
}
