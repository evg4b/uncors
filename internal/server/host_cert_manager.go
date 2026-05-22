package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/spf13/afero"
)

// HostCertManager manages TLS certificates for HTTPS mappings, generating a
// certificate per host on the fly signed by the local development CA.
type HostCertManager struct {
	fs        afero.Fs
	generator *CertGenerator
	cache     map[string]*tls.Certificate
	mutex     sync.RWMutex
}

func NewHostCertManager(fs afero.Fs) *HostCertManager {
	return &HostCertManager{
		fs:    fs,
		cache: make(map[string]*tls.Certificate),
	}
}

func (m *HostCertManager) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	err := m.ensureCA()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	host, ok := extractServerHost(clientHello)
	if !ok {
		return nil, ErrNoSNIProvided
	}

	cert, err := m.certificateForHost(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for %s: %w", host, err)
	}

	return cert, nil
}

// ensureCA lazily loads the default CA certificate and initialises the generator.
func (m *HostCertManager) ensureCA() error {
	m.mutex.RLock()

	if m.generator != nil {
		m.mutex.RUnlock()

		return nil
	}

	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.generator != nil {
		return nil
	}

	caCert, caKey, err := LoadDefaultCA(m.fs)
	if err != nil {
		return fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	err = CheckCAExpiration(caCert)
	if err != nil {
		return fmt.Errorf("CA certificate validation failed: %w", err)
	}

	m.generator = NewCertGenerator(caCert, caKey)

	return nil
}

// certificateForHost returns a cached certificate for the host, generating and
// caching a new one when it does not yet exist.
func (m *HostCertManager) certificateForHost(host string) (*tls.Certificate, error) {
	m.mutex.RLock()

	if cert, exists := m.cache[host]; exists {
		m.mutex.RUnlock()

		return cert, nil
	}

	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cert, exists := m.cache[host]; exists {
		return cert, nil
	}

	cert, err := m.generator.GenerateCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for %s: %w", host, err)
	}

	m.cache[host] = cert
	log.Printf("Generated TLS certificate for host: %s", host)

	return cert, nil
}

func extractServerHost(clientHello *tls.ClientHelloInfo) (string, bool) {
	if clientHello == nil {
		return "", false
	}

	if clientHello.ServerName != "" {
		return clientHello.ServerName, true
	}

	if clientHello.Conn == nil || clientHello.Conn.LocalAddr() == nil {
		return "", false
	}

	host, _, err := net.SplitHostPort(clientHello.Conn.LocalAddr().String())
	if err != nil {
		return "", false
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		return "", false
	}

	return host, true
}
