package tls

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

const (
	expirationWarningThreshold = 7 * 24 * time.Hour // 7 days
	hoursInDay                 = 24
)

// CertManager manages TLS certificates for HTTPS mappings.
type CertManager struct {
	generator *CertGenerator
	cache     map[string]*tls.Certificate
	mutex     sync.RWMutex
}

// NewCertManager creates a new certificate manager.
// If caCert and caKey are provided, it enables auto-generation.
func NewCertManager(caCert *x509.Certificate, caKey *rsa.PrivateKey) *CertManager {
	var generator *CertGenerator
	if caCert != nil && caKey != nil {
		generator = NewCertGenerator(caCert, caKey)
	}

	return &CertManager{
		generator: generator,
		cache:     make(map[string]*tls.Certificate),
	}
}

// GetCertificate returns a certificate for the given host.
// If auto-generation is enabled and no certificate exists, it generates a new one.
func (m *CertManager) GetCertificate(host string) (*tls.Certificate, error) {
	// Check cache first
	m.mutex.RLock()
	if cert, exists := m.cache[host]; exists {
		m.mutex.RUnlock()

		return cert, nil
	}
	m.mutex.RUnlock()

	// Generate new certificate if generator is available
	if m.generator == nil {
		return nil, ErrNoCertificateAvailable
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if cert, exists := m.cache[host]; exists {
		return cert, nil
	}

	// Generate and cache the certificate
	cert, err := m.generator.GenerateCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for %s: %w", host, err)
	}

	m.cache[host] = cert
	log.Debugf("Generated TLS certificate for host: %s", host)

	return cert, nil
}

// CheckCAExpiration checks if the CA certificate is expiring soon and logs a warning.
func CheckCAExpiration(cert *x509.Certificate) {
	expiresSoon, timeLeft := CheckExpiration(cert, expirationWarningThreshold)
	if expiresSoon {
		days := int(timeLeft.Hours() / hoursInDay)
		log.Warnf("CA certificate expires in %d days! Consider regenerating it with: uncors generate-certs --force", days)
	}
}
