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

const UncorsTestVersion = "v1.2.3"

func SetupBin(_ *testing.M) {
	compile.Do(func() {
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
	t.Helper()

	return exec.CommandContext(t.Context(), bin, args...)
}

func repoRootPath() string {
	out, err := exec.CommandContext(context.Background(), "go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		panic(fmt.Sprintf("failed to determine repository root: %v", err))
	}

	return strings.TrimSpace(string(out))
}
