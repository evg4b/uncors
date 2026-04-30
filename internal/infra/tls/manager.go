package tls

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
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
	output    contracts.Output
}

type CertManagerOptions = func(*CertManager)

func WithOutput(output contracts.Output) CertManagerOptions {
	return func(m *CertManager) {
		m.output = output
	}
}

func WithCert(caCert *x509.Certificate, caKey *rsa.PrivateKey) CertManagerOptions {
	return func(m *CertManager) {
		if caCert != nil && caKey != nil {
			m.generator = NewCertGenerator(caCert, caKey)
		}
	}
}

// NewCertManager creates a new certificate manager.
// If caCert and caKey are provided, it enables auto-generation.
func NewCertManager(options ...CertManagerOptions) *CertManager {
	return helpers.ApplyOptions(&CertManager{
		cache: make(map[string]*tls.Certificate),
	}, options)
}

// GetCertificate returns a certificate for the given host.
// If auto-generation is enabled and no certificate exists, it generates a new one.
func (m *CertManager) GetCertificate(host string) (*tls.Certificate, error) {
	m.mutex.RLock()

	if cert, exists := m.cache[host]; exists {
		m.mutex.RUnlock()

		return cert, nil
	}

	m.mutex.RUnlock()

	if m.generator == nil {
		return nil, ErrNoCertificateAvailable
	}

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

// CheckCAExpiration checks if the CA certificate is expiring soon and logs a warning.
func CheckCAExpiration(cert *x509.Certificate) error {
	expiresSoon, timeLeft := CheckExpiration(cert, expirationWarningThreshold)
	if !expiresSoon {
		return nil
	}

	switch {
	case timeLeft < 0:
		return ErrCACertExpired
	case timeLeft < 24*time.Hour:
		hours := int(timeLeft.Hours())

		return fmt.Errorf("CA certificate expires in less than %d hours! %w", hours, ErrCACertExpiringSoon)
	default:
		days := int(timeLeft.Hours() / hoursInDay)

		return fmt.Errorf("CA certificate expires in %d days! %w", days, ErrCACertExpiringSoon)
	}
}
