package validators

import "strings"

// Errors collects validation error messages.
type Errors []string

func (e *Errors) add(msg string) {
	*e = append(*e, msg)
}

func (e Errors) Error() string {
	return strings.Join(e, "\n")
}

func (e Errors) HasAny() bool {
	return len(e) > 0
}
