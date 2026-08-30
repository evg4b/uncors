package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	bin     string
	compile sync.Once
)

var (
	repoRoot     string
	repoRootOnce sync.Once
)

const UncorsTestVersion = "v1.2.3"

// SetupBin builds the uncors binary once per test binary. Call it from TestMain
// of any package that drives uncors as a subprocess.
func SetupBin(_ *testing.M) {
	compile.Do(func() {
		//nolint:usetesting // intentional: binary lifetime must span all tests, not one subtest
		tmp, err := os.MkdirTemp("", "uncors-test-*")
		if err != nil {
			panic(err)
		}

		bin = filepath.Join(tmp, "uncors")
		cmd := exec.CommandContext(
			context.Background(),
			"go", "build",
			"-o", bin,
			"-ldflags", fmt.Sprintf("-s -w -X 'main.Version=%s'", UncorsTestVersion),
			repoRootPath(),
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("failed to build uncors: %v\n%s", err, out))
		}
	})
}

func UncorsCommand(t *testing.T, args []string) *exec.Cmd {
	return exec.CommandContext(t.Context(), bin, args...)
}

func repoRootPath() string {
	repoRootOnce.Do(func() {
		out, err := exec.CommandContext(context.Background(), "go", "list", "-m", "-f", "{{.Dir}}").Output()
		if err != nil {
			panic(fmt.Sprintf("failed to determine repository root: %v", err))
		}

		repoRoot = strings.TrimSpace(string(out))
	})

	return repoRoot
}

func RepoRoot(t *testing.T) string {
	t.Helper()

	return repoRootPath()
}
