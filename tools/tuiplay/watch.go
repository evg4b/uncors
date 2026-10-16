package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	watchDebounce = 200 * time.Millisecond
	stopTimeout   = 2 * time.Second
)

// watchedDirs are the module directories whose Go files can change a story.
var watchedDirs = []string{"internal", "pkg", "tools"}

var errNoModule = errors.New("go.mod not found")

// watch shows the story in a child tuiplay built from the current sources,
// and rebuilds and restarts it whenever a Go file changes. It returns when
// the story is quit or tuiplay is interrupted.
func watch(story string) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}

	changes, closeWatcher, err := watchGoFiles(root)
	if err != nil {
		return err
	}

	defer closeWatcher()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	defer signal.Stop(signals)

	// A fresh private directory, so nobody else can plant the binary we run.
	buildDir, err := os.MkdirTemp("", "uncors-tuiplay-")
	if err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	defer os.RemoveAll(buildDir)

	restartOnChange(filepath.Join(buildDir, "tuiplay"), root, story, changes, signals)

	return nil
}

// restartOnChange runs the story until it is quit, restarting it on every
// change.
func restartOnChange(binary, root, story string, changes <-chan string, signals <-chan os.Signal) {
	ctx := context.Background()

	for {
		drain(changes)

		running := startStory(ctx, root, binary, story)

		select {
		case err := <-running.done():
			if err == nil {
				return
			}

			logf("the story stopped: %v; waiting for changes", err)

			select {
			case <-changes:
			case <-signals:
				return
			}
		case file := <-changes:
			running.stop()
			logf("%s changed, rebuilding", relative(root, file))
		case <-signals:
			running.stop()

			return
		}
	}
}

type runningStory struct {
	cmd    *exec.Cmd
	exited chan error
}

// startStory builds tuiplay and runs it on the story. It returns nil, and
// prints why, when either step fails.
func startStory(ctx context.Context, root, binary, story string) *runningStory {
	// The developer's own go from PATH, as the Go docs advise for locating it.
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./tools/tuiplay") // NOSONAR
	build.Dir = root
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr

	err := build.Run()
	if err != nil {
		logf("build failed; waiting for changes")

		return nil
	}

	cmd := exec.CommandContext(ctx, binary, "-story", story, "-watch=false") //nolint:gosec // our own build
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		logf("cannot start the story: %v; waiting for changes", err)

		return nil
	}

	running := &runningStory{cmd: cmd, exited: make(chan error, 1)}

	go func() {
		running.exited <- cmd.Wait()
	}()

	return running
}

// done is nil, and so blocks forever, when no story is running.
func (r *runningStory) done() <-chan error {
	if r == nil {
		return nil
	}

	return r.exited
}

// stop asks the story to quit, so Bubble Tea restores the terminal, and kills
// it if it does not.
func (r *runningStory) stop() {
	if r == nil {
		return
	}

	err := r.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		_ = r.cmd.Process.Kill()
	}

	select {
	case <-r.exited:
	case <-time.After(stopTimeout):
		_ = r.cmd.Process.Kill()
		<-r.exited
	}
}

// watchGoFiles reports, after a short quiet period, a Go file that was
// written, created, removed, or renamed under watchedDirs.
func watchGoFiles(root string) (<-chan string, func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	for _, dir := range watchedDirs {
		_, err = os.Stat(filepath.Join(root, dir))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		err = filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || !entry.IsDir() {
				return err
			}

			return watcher.Add(path)
		})
		if err != nil {
			return nil, nil, errors.Join(watcher.Close(), fmt.Errorf("failed to watch %s: %w", dir, err))
		}
	}

	changes := make(chan string, 1)

	go forwardChanges(watcher, changes)

	return changes, func() { _ = watcher.Close() }, nil
}

func forwardChanges(watcher *fsnotify.Watcher, changes chan<- string) {
	var debounce *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if !isGoChange(event) {
				continue
			}

			if debounce != nil {
				debounce.Stop()
			}

			debounce = time.AfterFunc(watchDebounce, func() {
				select {
				case changes <- event.Name:
				default:
				}
			})
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func isGoChange(event fsnotify.Event) bool {
	return strings.HasSuffix(event.Name, ".go") &&
		event.Has(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename)
}

func drain(changes <-chan string) {
	for {
		select {
		case <-changes:
		default:
			return
		}
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		_, err = os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoModule
		}

		dir = parent
	}
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}

	return rel
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tuiplay: "+format+"\n", args...)
}
