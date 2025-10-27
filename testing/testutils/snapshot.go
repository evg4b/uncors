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
