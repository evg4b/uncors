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
	// Check if the mapping is HTTPS
	fromURL, err := v.Mapping.GetFromURL()
	if err != nil {
		return // URL validation is handled elsewhere
	}

	if fromURL.Scheme != "https" {
		return // Not HTTPS, no TLS validation needed
	}

	// Check if both cert-file and key-file are provided or both are empty
	hasCert := v.Mapping.CertFile != ""
	hasKey := v.Mapping.KeyFile != ""

	if hasCert != hasKey {
		errors.Add(v.Field, "both cert-file and key-file must be provided together")
		return
	}

	// If custom certificates are provided, validate they exist
	if hasCert && hasKey {
		if exists, err := afero.Exists(v.Fs, v.Mapping.CertFile); err != nil || !exists {
			errors.Add(joinPath(v.Field, "cert-file"), fmt.Sprintf("certificate file not found: %s", v.Mapping.CertFile))
		}
		if exists, err := afero.Exists(v.Fs, v.Mapping.KeyFile); err != nil || !exists {
			errors.Add(joinPath(v.Field, "key-file"), fmt.Sprintf("key file not found: %s", v.Mapping.KeyFile))
		}
		return
	}

	// If no custom certificates provided, check if CA exists for auto-generation
	if !infratls.CAExists() {
		errorMessage := formatTLSError(fromURL.Host)
		errors.Add(v.Field, errorMessage)
	}
}

func formatTLSError(host string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTPS mapping '%s' requires TLS certificates.\n\n", host))
	sb.WriteString("Please choose one of the following options:\n")
	sb.WriteString("  1. Provide custom certificates for this mapping:\n")
	sb.WriteString("     cert-file: /path/to/your/certificate.crt\n")
	sb.WriteString("     key-file: /path/to/your/private-key.key\n\n")
	sb.WriteString("  2. Generate a local CA certificate for automatic TLS:\n")
	sb.WriteString("     uncors generate-certs\n\n")
	sb.WriteString("After generating CA, you can add it to your system's trusted certificates.")
	return sb.String()
}
