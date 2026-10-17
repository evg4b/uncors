package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/testing/stories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunList(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, run([]string{"-list"}, &out))

	assert.Contains(t, out.String(), "tracker/many\n")
}

func TestRunRender(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, run([]string{"-render", "help/short", "-width", "40", "-plain"}, &out))

	assert.Equal(t, "? toggle help • r reload • q quit\n", out.String())
}

func TestRunUnknownStory(t *testing.T) {
	require.ErrorIs(t, run([]string{"-render", "nope"}, &bytes.Buffer{}), errUnknownStory)
	require.ErrorIs(t, run([]string{"-story", "nope", "-state", filepath.Join(t.TempDir(), "s.json")},
		&bytes.Buffer{}), errUnknownStory)
}

func key(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: []rune(text)[0]})
}

func special(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func TestPicker(t *testing.T) {
	t.Run("starts on the last story", func(t *testing.T) {
		chooser := newPicker(stories.Stories(), "tracker/many")

		chooser.Update(special(tea.KeyEnter))

		assert.Equal(t, "tracker/many", chooser.chosen)
	})

	t.Run("filters as you type", func(t *testing.T) {
		chooser := newPicker(stories.Stories(), "")

		for _, char := range "hist" {
			chooser.Update(key(string(char)))
		}

		chooser.Update(special(tea.KeyDown))
		view := ansi.Strip(chooser.View().Content)
		assert.Contains(t, view, "› history/overflow")
		assert.NotContains(t, view, "tracker")

		_, cmd := chooser.Update(special(tea.KeyEnter))
		require.NotNil(t, cmd)
		assert.Equal(t, "history/overflow", chooser.chosen)
		assert.Empty(t, chooser.View().Content)
	})

	t.Run("esc cancels", func(t *testing.T) {
		chooser := newPicker(stories.Stories(), "")

		_, cmd := chooser.Update(special(tea.KeyEscape))

		require.NotNil(t, cmd)
		assert.Empty(t, chooser.chosen)
	})
}

func TestViewer(t *testing.T) {
	catalog := stories.Stories()

	t.Run("shows the story with a status line", func(t *testing.T) {
		view := newViewer(catalog[findStory(catalog, "help/short")])
		view.Update(tea.WindowSizeMsg{Width: 60, Height: 20})

		content := ansi.Strip(view.View().Content)

		assert.Contains(t, content, "? toggle help • r reload • q quit")
		assert.Contains(t, content, "help/short · q quit")
	})

	t.Run("gives full-height stories the terminal minus the status line", func(t *testing.T) {
		view := newViewer(catalog[findStory(catalog, "app/started")])
		view.Update(tea.WindowSizeMsg{Width: 80, Height: 12})

		assert.Len(t, strings.Split(view.View().Content, "\n"), 12)
	})

	t.Run("sends keys to the story", func(t *testing.T) {
		view := newViewer(catalog[findStory(catalog, "help/short")])
		view.Update(tea.WindowSizeMsg{Width: 120, Height: 20})

		_, cmd := view.Update(key("?"))

		assert.Nil(t, cmd)
		assert.Contains(t, ansi.Strip(view.View().Content), "scroll up")
	})

	t.Run("q quits", func(t *testing.T) {
		view := newViewer(catalog[0])

		_, cmd := view.Update(key("q"))

		require.NotNil(t, cmd)
		assert.IsType(t, tea.QuitMsg{}, cmd())
	})
}

func TestWatchGoFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "widget")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	changes, closeWatcher, err := watchGoFiles(root)
	require.NoError(t, err)
	t.Cleanup(closeWatcher)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "widget.go"), []byte("package widget"), 0o600))

	select {
	case file := <-changes:
		assert.Equal(t, filepath.Join(dir, "widget.go"), file)
	case <-time.After(5 * time.Second):
		t.Fatal("no change reported")
	}
}

func TestModuleRoot(t *testing.T) {
	root, err := moduleRoot()

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(root, "go.mod"))
	assert.DirExists(t, filepath.Join(root, "tools", "tuiplay"))
}
