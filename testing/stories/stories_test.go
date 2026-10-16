package stories_test

import (
	"strconv"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/testing/stories"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var widths = []int{40, 80, 120}

// TestStories snapshots every story at several widths: first the plain text
// for readable diffs, then the same frame with its styles.
func TestStories(t *testing.T) {
	for _, story := range stories.Stories() {
		t.Run(story.Name, func(t *testing.T) {
			for _, width := range widths {
				t.Run(strconv.Itoa(width), func(t *testing.T) {
					frame := stories.Render(story, width)

					testutils.MatchSnapshot(t, ansi.Strip(frame), frame)
				})
			}
		})
	}
}

func TestStoryNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}

	for _, story := range stories.Stories() {
		require.NotEmpty(t, story.Name)
		assert.False(t, seen[story.Name], "duplicate story %q", story.Name)

		seen[story.Name] = true
	}
}

func TestFind(t *testing.T) {
	story, ok := stories.Find("tracker/many")
	require.True(t, ok)
	assert.Equal(t, "tracker/many", story.Name)

	_, ok = stories.Find("nope")
	assert.False(t, ok)
}
