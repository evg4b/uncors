package config

import (
	"fmt"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type TLSError struct {
	Host string
}

func (e *TLSError) Error() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "HTTPS mapping '%s' requires a local CA certificate for automatic TLS.\n\n", e.Host)
	builder.WriteString("Generate a local CA certificate:\n")
	builder.WriteString("  uncors generate-certs\n\n")
	builder.WriteString("After generating CA, you can add it to your system's trusted certificates.")

	return builder.String()
}

func joinErrors(errs *multierror.Error) error {
	if errs == nil {
		return nil
	}

	errs.ErrorFormat = func(errs []error) string {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}

		return strings.Join(msgs, "\n")
	}

	return errs.ErrorOrNil()
}
