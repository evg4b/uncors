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
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		assert.NotNil(t, history.file)
		assert.NotNil(t, history.data)
		assert.Equal(t, int64(historyInitialSize), history.capacity)
		assert.Zero(t, history.writePos)
	})

	t.Run("removes file on close", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		name := history.file.Name()
		_, statErr := os.Stat(name)
		require.NoError(t, statErr, "file should exist before Close")

		require.NoError(t, history.Close())

		_, statErr = os.Stat(name)
		assert.True(t, os.IsNotExist(statErr), "temp file should be removed after Close")
	})
}

func TestHistory_AppendLine(t *testing.T) {
	t.Run("stores a single line", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("hello world")

		assert.Equal(t, 1, history.LineCount())
		assert.Equal(t, []string{"hello world"}, history.Lines())
	})

	t.Run("appends multiple independent lines", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("first")
		history.AppendLine("second")
		history.AppendLine("third")

		assert.Equal(t, 3, history.LineCount())
		assert.Equal(t, []string{"first", "second", "third"}, history.Lines())
	})

	t.Run("splits embedded newlines into separate viewport rows", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("line A\nline B\nline C")

		assert.Equal(t, 3, history.LineCount())
		assert.Equal(t, []string{"line A", "line B", "line C"}, history.Lines())
	})

	t.Run("strips trailing newlines before splitting", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("alpha\nbeta\n")

		// TrimRight removes the trailing \n, then split yields ["alpha", "beta"]
		assert.Equal(t, 2, history.LineCount())
		assert.Equal(t, []string{"alpha", "beta"}, history.Lines())
	})

	t.Run("stores empty line", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("")

		assert.Equal(t, 1, history.LineCount())
		assert.Equal(t, []string{""}, history.Lines())
	})

	t.Run("stores ANSI-styled strings", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		styled := "\x1b[31mred text\x1b[0m"
		history.AppendLine(styled)

		lines := history.Lines()
		require.Len(t, lines, 1)
		assert.Equal(t, styled, lines[0])
	})
}

func TestHistory_Grow(t *testing.T) {
	t.Run("grows capacity when data exceeds initial size", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		initialCap := history.capacity

		// Write enough 1-KB lines to fill the initial 1-MB mmap.
		longLine := strings.Repeat("x", 1024)

		count := int(initialCap/1024) + 10
		for range count {
			history.AppendLine(longLine)
		}

		assert.Greater(t, history.capacity, initialCap, "capacity should have grown")
		assert.Equal(t, count, history.LineCount())
	})

	t.Run("all lines remain readable after grow", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		longLine := strings.Repeat("y", 1024)

		count := int(history.capacity/1024) + 5
		for range count {
			history.AppendLine(longLine)
		}

		for _, line := range history.Lines() {
			assert.Equal(t, longLine, line)
		}
	})
}

func TestHistory_LineCount(t *testing.T) {
	t.Run("returns zero for empty history", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		assert.Zero(t, history.LineCount())
	})

	t.Run("increments with each appended line", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		for i := 1; i <= 5; i++ {
			history.AppendLine("line")
			assert.Equal(t, i, history.LineCount())
		}
	})

	t.Run("counts sub-lines from multi-line input", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("a\nb\nc")

		assert.Equal(t, 3, history.LineCount())
	})
}

func TestHistory_Lines(t *testing.T) {
	t.Run("returns nil for empty history", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		assert.Empty(t, history.Lines())
	})

	t.Run("returned slice grows as lines are appended", func(t *testing.T) {
		history, err := newHistory()
		require.NoError(t, err)

		defer history.Close()

		history.AppendLine("one")
		assert.Len(t, history.Lines(), 1)

		history.AppendLine("two")
		assert.Len(t, history.Lines(), 2)
	})
}
