package integration

import (
	"fmt"
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

const UncorsTestVrsion = "v1.2.3"

func UncorsCommand(t *testing.T, args []string) *exec.Cmd {
	compile.Do(func() {
		tmp := t.TempDir()
		bin = filepath.Join(tmp, "uncors")
		cmd := exec.Command(
			"go",
			"build",
			"-o", bin,
			"-ldflags",
			fmt.Sprintf("-s -w -X 'main.Version=%s'", UncorsTestVrsion),
			RepoRoot(t),
		)
		_, err := cmd.CombinedOutput()
		if err != nil {
			panic(err)
		}
	})

	return exec.CommandContext(t.Context(), bin, args...)
}

func RepoRoot(t *testing.T) string {

	t.Helper()
	repoRootOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
		if err != nil {
			t.Fatalf("failed to determine repository root: %v", err)
		}
		repoRoot = strings.TrimSpace(string(out))
	})
	return repoRoot

}
