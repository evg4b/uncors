package sfmt

import (
	"fmt"
	"io"
)

func Fprint(w io.Writer, payload ...any) {
	_, err := fmt.Fprint(w, payload...)
	if err != nil {
		panic(err)
	}
}

func Fprintf(w io.Writer, format string, a ...any) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		panic(err)
	}
}

func Sprintf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

func Fprintln(w io.Writer, a ...any) {
	_, err := fmt.Fprintln(w, a...)
	if err != nil {
		panic(err)
	}
}
