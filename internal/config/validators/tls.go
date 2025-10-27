package validators

import (
	"fmt"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
)

type TLSValidator struct {
	Field   string
	Mapping config.Mapping
	Fs      afero.Fs
}

func (v *TLSValidator) IsValid(errors *validate.Errors) {
	fromURL, err := v.Mapping.GetFromURL()
	if err != nil {
		return // URL validation is handled elsewhere
	}

	if fromURL.Scheme != "https" {
		return // Not HTTPS, no TLS validation needed
	}

	// Check if CA exists for auto-generation
	v.validateCAAvailability(errors, fromURL.Host)
}

func (v *TLSValidator) validateCAAvailability(errors *validate.Errors, host string) {
	if !infratls.CAExists(v.Fs) {
		errorMessage := formatTLSError(host)
		errors.Add(v.Field, errorMessage)
	}
}

func formatTLSError(host string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("HTTPS mapping '%s' requires a local CA certificate for automatic TLS.\n\n", host))
	builder.WriteString("Generate a local CA certificate:\n")
	builder.WriteString("  uncors generate-certs\n\n")
	builder.WriteString("After generating CA, you can add it to your system's trusted certificates.")

	return builder.String()
}
