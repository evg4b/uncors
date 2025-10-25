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
	// Map host -> certificate configuration
	certMap := make(map[string]certConfig)
	hasAutoGen := false

	for _, mapping := range mappings {
		host, _, err := mapping.GetFromHostPort()
		if err != nil {
			return nil, fmt.Errorf("failed to parse mapping host: %w", err)
		}

		if mapping.CertFile != "" && mapping.KeyFile != "" {
			// Custom certificate specified
			certMap[host] = certConfig{
				certFile: mapping.CertFile,
				keyFile:  mapping.KeyFile,
			}
		} else {
			// Will use auto-generated certificate
			hasAutoGen = true
		}
	}

	// Load CA for auto-generation if needed
	var certManager *infratls.CertManager
	if hasAutoGen {
		caCert, caKey, err := infratls.LoadDefaultCA()
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
		}

		// Check CA expiration
		infratls.CheckCAExpiration(caCert)

		certManager = infratls.NewCertManager(caCert, caKey)
	}

	// Create TLS config with GetCertificate callback
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// Check if custom certificate is specified for this host
			if cfg, exists := certMap[hello.ServerName]; exists {
				cert, err := tls.LoadX509KeyPair(cfg.certFile, cfg.keyFile)
				if err != nil {
					return nil, fmt.Errorf("failed to load custom certificate for %s: %w", hello.ServerName, err)
				}
				return &cert, nil
			}

			// Use auto-generated certificate
			if certManager != nil {
				return certManager.GetCertificate(hello.ServerName)
			}

			return nil, fmt.Errorf("no certificate available for %s", hello.ServerName)
		},
	}

	return tlsConfig, nil
}

type certConfig struct {
	certFile string
	keyFile  string
}
