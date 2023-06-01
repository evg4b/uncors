package sfmt

import (
	"fmt"
	"io"
)

func Fprint(w io.Writer, payload ...any) {
	if _, err := fmt.Fprint(w, payload...); err != nil {
		panic(err)
	}
}

func Fprintf(w io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		panic(err)
	}
}

func Sprintf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

func Fprintln(w io.Writer, a ...any) {
	if _, err := fmt.Fprintln(w, a...); err != nil {
		panic(err)
	}
}
