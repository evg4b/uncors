package uncorsapp

import (
	"strconv"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storyWidths = []int{40, 80, 120}

// TestStories snapshots every story at several widths: first the plain text
// for readable diffs, then the same frame with its styles.
func TestStories(t *testing.T) {
	for _, story := range Stories() {
		t.Run(story.Name, func(t *testing.T) {
			for _, width := range storyWidths {
				t.Run(strconv.Itoa(width), func(t *testing.T) {
					frame := RenderStory(story, width)

					testutils.MatchSnapshot(t, ansi.Strip(frame), frame)
				})
			}
		})
	}
}

func TestStoryNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}

	for _, story := range Stories() {
		require.NotEmpty(t, story.Name)
		assert.False(t, seen[story.Name], "duplicate story %q", story.Name)

		seen[story.Name] = true
	}
}
