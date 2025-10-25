package uncors

import (
	"crypto/tls"
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

// buildTLSConfig creates a TLS configuration for the given HTTPS mappings.
// It supports both custom certificates per mapping and auto-generated certificates.
func buildTLSConfig(fs afero.Fs, mappings config.Mappings) (*tls.Config, error) {
	// Since all mappings in a port group share the same port, we can use the first mapping's certificate
	// For SNI to work properly with custom certificates, all mappings should use the same certificate
	// or we need to match by host. For now, we'll use the first mapping's certificate configuration.

	if len(mappings) == 0 {
		return nil, infratls.ErrNoMappingsProvided
	}

	firstMapping := mappings[0]
	hasCustomCert := firstMapping.CertFile != "" && firstMapping.KeyFile != ""

	// If custom certificate is provided, load it once and reuse
	if hasCustomCert {
		certData, err := afero.ReadFile(fs, firstMapping.CertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file: %w", err)
		}
		keyData, err := afero.ReadFile(fs, firstMapping.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}

		return &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}, nil
	}

	// Load CA for auto-generation
	caCert, caKey, err := infratls.LoadDefaultCA()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	// Check CA expiration
	infratls.CheckCAExpiration(caCert)

	certManager := infratls.NewCertManager(caCert, caKey)

	// Extract host from first mapping for certificate generation
	host, _, err := firstMapping.GetFromHostPort()
	if err != nil {
		return nil, fmt.Errorf("failed to parse mapping host: %w", err)
	}

	cert, err := certManager.GetCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*cert},
	}, nil
}
