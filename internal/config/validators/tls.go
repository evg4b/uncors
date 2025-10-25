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

	hasCert := v.Mapping.CertFile != ""
	hasKey := v.Mapping.KeyFile != ""

	if hasCert != hasKey {
		errors.Add(v.Field, "both cert-file and key-file must be provided together")

		return
	}

	// If custom certificates are provided, validate they exist
	if hasCert && hasKey {
		v.validateCustomCertificates(errors)

		return
	}

	// If no custom certificates provided, check if CA exists for auto-generation
	v.validateCAAvailability(errors, fromURL.Host)
}

func (v *TLSValidator) validateCustomCertificates(errors *validate.Errors) {
	if exists, err := afero.Exists(v.Fs, v.Mapping.CertFile); err != nil || !exists {
		errors.Add(joinPath(v.Field, "cert-file"), fmt.Sprintf("certificate file not found: %s", v.Mapping.CertFile))
	}
	if exists, err := afero.Exists(v.Fs, v.Mapping.KeyFile); err != nil || !exists {
		errors.Add(joinPath(v.Field, "key-file"), fmt.Sprintf("key file not found: %s", v.Mapping.KeyFile))
	}
}

func (v *TLSValidator) validateCAAvailability(errors *validate.Errors, host string) {
	if !infratls.CAExists(v.Fs) {
		errorMessage := formatTLSError(host)
		errors.Add(v.Field, errorMessage)
	}
}

func formatTLSError(host string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("HTTPS mapping '%s' requires TLS certificates.\n\n", host))
	builder.WriteString("Please choose one of the following options:\n")
	builder.WriteString("  1. Provide custom certificates for this mapping:\n")
	builder.WriteString("     cert-file: /path/to/your/certificate.crt\n")
	builder.WriteString("     key-file: /path/to/your/private-key.key\n\n")
	builder.WriteString("  2. Generate a local CA certificate for automatic TLS:\n")
	builder.WriteString("     uncors generate-certs\n\n")
	builder.WriteString("After generating CA, you can add it to your system's trusted certificates.")

	return builder.String()
}
