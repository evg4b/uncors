package uncorsapp

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHistory(t *testing.T) {
	t.Run("creates temp file and mmap", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		assert.NotNil(t, h.file)
		assert.NotNil(t, h.data)
		assert.Equal(t, int64(historyInitialSize), h.capacity)
		assert.Zero(t, h.writePos)
	})

	t.Run("removes file on close", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		name := h.file.Name()
		_, statErr := os.Stat(name)
		require.NoError(t, statErr, "file should exist before Close")

		require.NoError(t, h.Close())

		_, statErr = os.Stat(name)
		assert.True(t, os.IsNotExist(statErr), "temp file should be removed after Close")
	})
}

func TestHistory_AppendLine(t *testing.T) {
	t.Run("stores a single line", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("hello world")

		assert.Equal(t, 1, h.LineCount())
		assert.Equal(t, []string{"hello world"}, h.Lines())
	})

	t.Run("appends multiple independent lines", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("first")
		h.AppendLine("second")
		h.AppendLine("third")

		assert.Equal(t, 3, h.LineCount())
		assert.Equal(t, []string{"first", "second", "third"}, h.Lines())
	})

	t.Run("splits embedded newlines into separate viewport rows", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("line A\nline B\nline C")

		assert.Equal(t, 3, h.LineCount())
		assert.Equal(t, []string{"line A", "line B", "line C"}, h.Lines())
	})

	t.Run("strips trailing newlines before splitting", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("alpha\nbeta\n")

		// TrimRight removes the trailing \n, then split yields ["alpha", "beta"]
		assert.Equal(t, 2, h.LineCount())
		assert.Equal(t, []string{"alpha", "beta"}, h.Lines())
	})

	t.Run("stores empty line", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("")

		assert.Equal(t, 1, h.LineCount())
		assert.Equal(t, []string{""}, h.Lines())
	})

	t.Run("stores ANSI-styled strings", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		styled := "\x1b[31mred text\x1b[0m"
		h.AppendLine(styled)

		lines := h.Lines()
		require.Len(t, lines, 1)
		assert.Equal(t, styled, lines[0])
	})
}

func TestHistory_Grow(t *testing.T) {
	t.Run("grows capacity when data exceeds initial size", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		initialCap := h.capacity

		// Write enough 1-KB lines to fill the initial 1-MB mmap.
		longLine := strings.Repeat("x", 1024)

		count := int(initialCap/1024) + 10
		for range count {
			h.AppendLine(longLine)
		}

		assert.Greater(t, h.capacity, initialCap, "capacity should have grown")
		assert.Equal(t, count, h.LineCount())
	})

	t.Run("all lines remain readable after grow", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		longLine := strings.Repeat("y", 1024)

		count := int(h.capacity/1024) + 5
		for range count {
			h.AppendLine(longLine)
		}

		for _, line := range h.Lines() {
			assert.Equal(t, longLine, line)
		}
	})
}

func TestHistory_LineCount(t *testing.T) {
	t.Run("returns zero for empty history", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		assert.Zero(t, h.LineCount())
	})

	t.Run("increments with each appended line", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		for i := 1; i <= 5; i++ {
			h.AppendLine("line")
			assert.Equal(t, i, h.LineCount())
		}
	})

	t.Run("counts sub-lines from multi-line input", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("a\nb\nc")

		assert.Equal(t, 3, h.LineCount())
	})
}

func TestHistory_Lines(t *testing.T) {
	t.Run("returns nil for empty history", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		assert.Empty(t, h.Lines())
	})

	t.Run("returned slice grows as lines are appended", func(t *testing.T) {
		h, err := newHistory()
		require.NoError(t, err)

		defer h.Close()

		h.AppendLine("one")
		assert.Len(t, h.Lines(), 1)

		h.AppendLine("two")
		assert.Len(t, h.Lines(), 2)
	})
}
