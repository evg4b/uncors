package uncorsapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelWriter(t *testing.T) {
	t.Run("forwards a rendered line without its trailing newline", func(t *testing.T) {
		lines := make(chan string, 1)

		count, err := newChannelWriter(lines).Write([]byte("a line\n"))

		require.NoError(t, err)
		assert.Equal(t, len("a line\n"), count, "the writer must report every byte consumed")
		assert.Equal(t, "a line", <-lines)
	})

	t.Run("drops empty writes", func(t *testing.T) {
		lines := make(chan string, 1)

		_, err := newChannelWriter(lines).Write([]byte("\n"))

		require.NoError(t, err)
		assert.Empty(t, lines, "a bare newline carries nothing to show")
	})

	// Presentation must never stall whatever produced the line.
	t.Run("never blocks when the model cannot keep up", func(t *testing.T) {
		lines := make(chan string, 1)
		writer := newChannelWriter(lines)

		for range 100 {
			_, err := writer.Write([]byte("overflow\n"))
			require.NoError(t, err)
		}

		assert.Len(t, lines, 1)
	})
}
