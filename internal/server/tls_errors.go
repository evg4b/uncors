package server

import (
	"fmt"
	"strings"
)

// TLSError explains that an https mapping cannot be served because the local CA
// has not been generated yet. The check lives here rather than in config
// validation: whether the CA is on disk is a property of the machine, not of the
// configuration file.
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
