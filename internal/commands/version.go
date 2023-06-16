package commands

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/evg4b/uncors/internal/sfmt"
)

func version(ctx context.Context, args []string, cwd string) {
	version, ok := ctx.Value("version").(string)
	if !ok {
		panic("ctx variable version is not defined")
	}

	sfmt.Fprintln(os.Stdout, fmt.Sprintf("uncors %s (%s:%s)", version, runtime.GOOS, runtime.GOARCH))
}
