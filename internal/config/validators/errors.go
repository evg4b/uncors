package validators

import "strings"

// Errors collects validation error messages.
//
//nolint:recvcheck // Error/HasAny need value receivers so Errors satisfies the error interface as a value type
type Errors []string

func (e Errors) Error() string {
	return strings.Join(e, "\n")
}

func (e Errors) HasAny() bool {
	return len(e) > 0
}

// add appends a validation message. Uses a pointer receiver because it mutates the slice.
func (e *Errors) add(msg string) {
	*e = append(*e, msg)
}
