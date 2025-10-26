package uncors

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

// hostCertManager manages certificates for different hosts using SNI.
type hostCertManager struct {
	fs          afero.Fs
	mappings    config.Mappings
	customCerts map[string]*tls.Certificate // host -> custom certificate
	certManager *infratls.CertManager       // for auto-generated certificates
	mutex       sync.RWMutex
}

// newHostCertManager creates a new host-based certificate manager.
func newHostCertManager(fs afero.Fs, mappings config.Mappings) (*hostCertManager, error) {
	manager := &hostCertManager{
		fs:          fs,
		mappings:    mappings,
		customCerts: make(map[string]*tls.Certificate),
	}

	// Check if we need CA for auto-generation
	needsCA := false
	for _, mapping := range mappings {
		hasCustomCert := mapping.CertFile != "" && mapping.KeyFile != ""
		if !hasCustomCert {
			needsCA = true

			break
		}
	}

	// Load CA if needed for auto-generation
	if needsCA {
		caCert, caKey, err := infratls.LoadDefaultCA(fs)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
		}
		infratls.CheckCAExpiration(caCert)
		manager.certManager = infratls.NewCertManager(caCert, caKey)
	}

	// Load custom certificates for each mapping
	for _, mapping := range mappings {
		hasCustomCert := mapping.CertFile != "" && mapping.KeyFile != ""
		if !hasCustomCert {
			continue
		}

		host, _, err := mapping.GetFromHostPort()
		if err != nil {
			return nil, fmt.Errorf("failed to parse mapping host: %w", err)
		}

		certData, err := afero.ReadFile(fs, mapping.CertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file for %s: %w", host, err)
		}
		keyData, err := afero.ReadFile(fs, mapping.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file for %s: %w", host, err)
		}
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate for %s: %w", host, err)
		}

		manager.customCerts[host] = &cert
		log.Debugf("Loaded custom certificate for host: %s", host)
	}

	return manager, nil
}

// getFallbackHost returns the first available host when no SNI is provided.
func (m *hostCertManager) getFallbackHost() (string, error) {
	// Try to use first custom certificate if available
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.customCerts) > 0 {
		for fallbackHost := range m.customCerts {
			log.Debugf("No SNI provided, using fallback host from custom certs: %s", fallbackHost)

			return fallbackHost, nil
		}
	}

	// Otherwise, use first mapping host
	if len(m.mappings) > 0 {
		firstHost, _, err := m.mappings[0].GetFromHostPort()
		if err == nil {
			log.Debugf("No SNI provided, using fallback host from mappings: %s", firstHost)

			return firstHost, nil
		}
	}

	return "", infratls.ErrNoSNIAndNoFallback
}

// getCustomCertificate returns a custom certificate for the given host if available.
func (m *hostCertManager) getCustomCertificate(host string) (*tls.Certificate, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cert, exists := m.customCerts[host]

	return cert, exists
}

// getCertificate implements SNI by selecting the appropriate certificate based on the requested host.
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

	// Try to get custom certificate for the requested host
	if cert, exists := m.getCustomCertificate(host); exists {
		return cert, nil
	}

	// If no custom certificate and no auto-generation available
	if m.certManager == nil {
		return nil, fmt.Errorf("%w: %s", infratls.ErrNoCertificateForHost, host)
	}

	// Auto-generate certificate for this host
	cert, err := m.certManager.GetCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for %s: %w", host, err)
	}

	return cert, nil
}

// buildTLSConfig creates a TLS configuration for the given HTTPS mappings.
// It supports both custom certificates per mapping and auto-generated certificates using SNI.
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
