package uncorsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHistory(t *testing.T) {
	t.Run("creates history", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		assert.NotNil(t, history.lines)
		assert.Equal(t, 0, history.LineCount())
	})
}

func TestHistory_AppendLine(t *testing.T) {
	t.Run("stores a single line", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("hello world")

		assert.Equal(t, 1, history.LineCount())
		assert.Equal(t, []string{"hello world"}, history.Lines())
	})

	t.Run("appends multiple independent lines", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("first")
		history.AppendLine("second")
		history.AppendLine("third")

		assert.Equal(t, 3, history.LineCount())
		assert.Equal(t, []string{"first", "second", "third"}, history.Lines())
	})

	t.Run("splits embedded newlines into separate viewport rows", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("line A\nline B\nline C")

		assert.Equal(t, 3, history.LineCount())
		assert.Equal(t, []string{"line A", "line B", "line C"}, history.Lines())
	})

	t.Run("strips trailing newlines before splitting", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("alpha\nbeta\n")

		assert.Equal(t, 2, history.LineCount())
		assert.Equal(t, []string{"alpha", "beta"}, history.Lines())
	})

	t.Run("stores empty line", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("")

		assert.Equal(t, 1, history.LineCount())
		assert.Equal(t, []string{""}, history.Lines())
	})

	t.Run("stores ANSI-styled strings", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		styled := "\x1b[31mred text\x1b[0m"
		history.AppendLine(styled)

		lines := history.Lines()
		require.Len(t, lines, 1)
		assert.Equal(t, styled, lines[0])
	})

	t.Run("respects historyMaxLines", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		longLine := "a"

		count := historyMaxLines + 5
		for range count {
			history.AppendLine(longLine)
		}

		assert.Equal(t, historyMaxLines, history.LineCount())
	})
}

func TestHistory_LineCount(t *testing.T) {
	t.Run("returns zero for empty history", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		assert.Zero(t, history.LineCount())
	})

	t.Run("increments with each appended line", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		for i := 1; i <= 5; i++ {
			history.AppendLine("line")
			assert.Equal(t, i, history.LineCount())
		}
	})

	t.Run("counts sub-lines from multi-line input", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("a\nb\nc")

		assert.Equal(t, 3, history.LineCount())
	})
}

func TestHistory_Lines(t *testing.T) {
	t.Run("returns empty slice for empty history", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		assert.Empty(t, history.Lines())
	})

	t.Run("returned slice grows as lines are appended", func(t *testing.T) {
		history := newHistory()

		defer history.Close()

		history.AppendLine("one")
		assert.Len(t, history.Lines(), 1)

		history.AppendLine("two")
		assert.Len(t, history.Lines(), 2)
	})
}
