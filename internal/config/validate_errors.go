package config

import "strings"

//nolint:recvcheck // Error/HasAny need value receivers so Errors satisfies the error interface as a value type
type Errors []string

func (e Errors) Error() string {
	return strings.Join(e, "\n")
}

func (e Errors) HasAny() bool {
	return len(e) > 0
}

func (e *Errors) add(msg string) {
	*e = append(*e, msg)
}
