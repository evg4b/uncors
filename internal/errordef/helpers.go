package errordef

import (
	"fmt"
	"io"
)

func printLnSafe(w io.Writer, values ...any) {
	if _, sysErr := fmt.Fprintln(w, values...); sysErr != nil {
		panic(sysErr)
	}
}
