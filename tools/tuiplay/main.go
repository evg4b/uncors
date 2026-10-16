// Command tuiplay shows one TUI story from testing/stories on its own,
// without starting the proxy.
//
//	go run ./tools/tuiplay                        # pick a story, show it, rebuild on every change
//	go run ./tools/tuiplay -story tracker/many    # skip the picker
//	go run ./tools/tuiplay -render tracker/many   # print one frame and exit
//	go run ./tools/tuiplay -list                  # print the story names
//
// While a story is shown, tuiplay watches the module's Go files, and on every
// change rebuilds itself and reopens the same story.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/testing/stories"
	"github.com/spf13/afero"
)

const (
	defaultWidth = 80
	statePerm    = 0o600
	stateDirPerm = 0o700
)

var errUnknownStory = errors.New("unknown story")

type options struct {
	story  string
	render string
	width  int
	plain  bool
	list   bool
	watch  bool
	state  string
}

func main() {
	err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuiplay:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	opts := options{}
	flags := flag.NewFlagSet("tuiplay", flag.ContinueOnError)
	flags.StringVar(&opts.story, "story", "", "story to show; without it tuiplay asks")
	flags.StringVar(&opts.render, "render", "", "print the first frame of this story and exit")
	flags.IntVar(&opts.width, "width", defaultWidth, "with -render: terminal width")
	flags.BoolVar(&opts.plain, "plain", false, "with -render: strip colours")
	flags.BoolVar(&opts.list, "list", false, "print the story names and exit")
	flags.BoolVar(&opts.watch, "watch", true, "rebuild and reopen the story when a Go file changes")
	flags.StringVar(&opts.state, "state", defaultStatePath(),
		"file that remembers the last story for the picker; empty to forget")

	err := flags.Parse(args)
	if err != nil {
		return err
	}

	catalog := stories.Stories()

	switch {
	case opts.list:
		for _, story := range catalog {
			fmt.Fprintln(out, story.Name)
		}

		return nil
	case opts.render != "":
		return render(out, catalog, opts)
	}

	return play(catalog, opts)
}

func play(catalog []stories.Story, opts options) error {
	fs := afero.NewOsFs()

	name := opts.story
	if name == "" {
		picked, err := pick(catalog, loadState(fs, opts.state).Story)
		if err != nil || picked == "" {
			return err
		}

		name = picked
	}

	index := findStory(catalog, name)
	if index < 0 {
		return fmt.Errorf("%w %q, see -list", errUnknownStory, name)
	}

	saveState(fs, opts.state, savedState{Story: name})

	if opts.watch {
		return watch(name)
	}

	return show(catalog[index])
}

func render(out io.Writer, catalog []stories.Story, opts options) error {
	index := findStory(catalog, opts.render)
	if index < 0 {
		return fmt.Errorf("%w %q, see -list", errUnknownStory, opts.render)
	}

	frame := stories.Render(catalog[index], opts.width)
	if opts.plain {
		frame = ansi.Strip(frame)
	}

	_, err := fmt.Fprintln(out, frame)

	return err
}

func findStory(catalog []stories.Story, name string) int {
	for i, story := range catalog {
		if story.Name == name {
			return i
		}
	}

	return -1
}

type savedState struct {
	Story string `json:"story"`
}

// defaultStatePath is in the user's own cache directory, not a shared temp
// directory, or "" when there is none.
func defaultStatePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}

	return filepath.Join(dir, "uncors", "tuiplay.json")
}

func loadState(fs afero.Fs, path string) savedState {
	state := savedState{}

	if path == "" {
		return state
	}

	data, err := afero.ReadFile(fs, path)
	if err == nil {
		_ = json.Unmarshal(data, &state)
	}

	return state
}

func saveState(fs afero.Fs, path string, state savedState) {
	if path == "" || fs.MkdirAll(filepath.Dir(path), stateDirPerm) != nil {
		return
	}

	data, err := json.Marshal(state)
	if err == nil {
		_ = afero.WriteFile(fs, path, data, statePerm)
	}
}
