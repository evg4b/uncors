package uncors

import (
	"crypto/tls"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

// hostCertManager manages auto-generated certificates for different hosts using SNI.
type hostCertManager struct {
	mappings    config.Mappings
	certManager *infratls.CertManager // for auto-generated certificates
}

// newHostCertManager creates a new host-based certificate manager.
func newHostCertManager(fs afero.Fs, mappings config.Mappings) (*hostCertManager, error) {
	// Load CA for auto-generation
	caCert, caKey, err := infratls.LoadDefaultCA(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	infratls.CheckCAExpiration(caCert)

	return &hostCertManager{
		mappings:    mappings,
		certManager: infratls.NewCertManager(caCert, caKey),
	}, nil
}

// getFallbackHost returns the first available host when no SNI is provided.
func (m *hostCertManager) getFallbackHost() (string, error) {
	if len(m.mappings) > 0 {
		firstHost, _, err := m.mappings[0].GetFromHostPort()
		if err == nil {
			log.Debugf("No SNI provided, using fallback host from mappings: %s", firstHost)

			return firstHost, nil
		}
	}

	return "", infratls.ErrNoSNIAndNoFallback
}

// getCertificate implements SNI by auto-generating certificates based on the requested host.
func (m *hostCertManager) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := clientHello.ServerName

	// If no SNI host provided, try to use fallback
	if host == "" {
		var err error

		host, err = m.getFallbackHost()
		if err != nil {
			return nil, err
		}
	}

	// Auto-generate certificate for this host
	cert, err := m.certManager.GetCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for %s: %w", host, err)
	}

	return cert, nil
}

// buildTLSConfig creates a TLS configuration for the given HTTPS mappings.
// It uses auto-generated certificates with SNI support.
func buildTLSConfig(fs afero.Fs, mappings config.Mappings) (*tls.Config, error) {
	if len(mappings) == 0 {
		return nil, infratls.ErrNoMappingsProvided
	}

	manager, err := newHostCertManager(fs, mappings)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: manager.getCertificate,
	}, nil
}
