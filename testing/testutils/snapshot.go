package testutils

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

const callerDepth = 2

var (
	projectRoot  string
	snapshotsDir string
	initOnce     sync.Once
)

// MatchSnapshot matches the given value against a snapshot.
func MatchSnapshot(t *testing.T, values ...any) {
	config := snaps.WithConfig(snaps.Dir(getSnapshotDir()))
	config.MatchSnapshot(t, values...)
}

// MatchJSON matches the given JSON value against a snapshot.
func MatchJSON(t *testing.T, value any) {
	config := snaps.WithConfig(snaps.Dir(getSnapshotDir()))
	config.MatchJSON(t, value)
}

// MatchYAML matches the given YAML value against a snapshot.
func MatchYAML(t *testing.T, value any) {
	config := snaps.WithConfig(snaps.Dir(getSnapshotDir()))
	config.MatchYAML(t, value)
}

func initDirs() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get snapshot.go file path")
	}

	projectRoot = filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	snapshotsDir = filepath.Join(projectRoot, "testing", "snapshots")
}

// getSnapshotDir returns the snapshot directory path based on the caller's location.
func getSnapshotDir() string {
	initOnce.Do(initDirs)

	_, testFile, _, ok := runtime.Caller(callerDepth)
	if !ok {
		panic("failed to get test file path")
	}

	relPath, err := filepath.Rel(projectRoot, filepath.Dir(testFile))
	if err != nil {
		panic("failed to get relative path: " + err.Error())
	}

	return filepath.Join(snapshotsDir, relPath)
}

// WithSnapshotConfig allows custom snapshot configuration.
func WithSnapshotConfig(options ...func(*snaps.Config)) *snaps.Config {
	opts := append([]func(*snaps.Config){snaps.Dir(getSnapshotDir())}, options...)

	return snaps.WithConfig(opts...)
}
