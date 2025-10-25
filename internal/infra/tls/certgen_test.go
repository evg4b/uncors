package tls_test

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertGenerator(t *testing.T) {
	t.Run("should create cert generator with valid CA", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		caCert, caKey, err := infratls.LoadCA(certPath, keyPath)
		require.NoError(t, err)

		generator := infratls.NewCertGenerator(caCert, caKey)
		assert.NotNil(t, generator)
	})
}

func TestCertGenerator_GenerateCertificate(t *testing.T) {
	// Setup: Create CA for all tests
	tmpDir := t.TempDir()
	config := infratls.CAConfig{
		ValidityDays: 365,
		OutputDir:    tmpDir,
	}
	certPath, keyPath, err := infratls.GenerateCA(config)
	require.NoError(t, err)

	caCert, caKey, err := infratls.LoadCA(certPath, keyPath)
	require.NoError(t, err)

	generator := infratls.NewCertGenerator(caCert, caKey)

	t.Run("should generate certificate for localhost", func(t *testing.T) {
		cert, err := generator.GenerateCertificate("localhost")
		require.NoError(t, err)
		assert.NotNil(t, cert)

		// Verify certificate has private key
		assert.NotNil(t, cert.PrivateKey)
		assert.NotEmpty(t, cert.Certificate)
	})

	t.Run("should generate certificate for custom domain", func(t *testing.T) {
		cert, err := generator.GenerateCertificate("example.local")
		require.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("should generate valid certificate with correct properties", func(t *testing.T) {
		host := "test.local"
		cert, err := generator.GenerateCertificate(host)
		require.NoError(t, err)

		// Parse the certificate
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Verify properties
		assert.Equal(t, host, x509Cert.Subject.CommonName)
		assert.Contains(t, x509Cert.DNSNames, host)
		assert.Contains(t, x509Cert.Subject.Organization, "UNCORS Development CA")

		// Verify key usage
		assert.NotEqual(t, 0, x509Cert.KeyUsage&x509.KeyUsageDigitalSignature)
		assert.NotEqual(t, 0, x509Cert.KeyUsage&x509.KeyUsageKeyEncipherment)
		assert.Contains(t, x509Cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

		// Verify it's not a CA
		assert.False(t, x509Cert.IsCA)

		// Verify validity period (approximately 365 days)
		expectedDuration := 365 * 24 * time.Hour
		actualDuration := x509Cert.NotAfter.Sub(x509Cert.NotBefore)
		tolerance := 5 * time.Minute
		assert.InDelta(t, expectedDuration, actualDuration, float64(tolerance))
	})

	t.Run("should generate certificate signed by CA", func(t *testing.T) {
		cert, err := generator.GenerateCertificate("verify.local")
		require.NoError(t, err)

		// Parse the generated certificate
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Verify it's signed by our CA
		roots := x509.NewCertPool()
		roots.AddCert(caCert)

		opts := x509.VerifyOptions{
			Roots:     roots,
			DNSName:   "verify.local",
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}

		_, err = x509Cert.Verify(opts)
		assert.NoError(t, err, "Certificate should be verifiable with CA")
	})

	t.Run("should generate unique certificates for different hosts", func(t *testing.T) {
		cert1, err := generator.GenerateCertificate("host1.local")
		require.NoError(t, err)

		cert2, err := generator.GenerateCertificate("host2.local")
		require.NoError(t, err)

		// Parse certificates
		x509Cert1, err := x509.ParseCertificate(cert1.Certificate[0])
		require.NoError(t, err)
		x509Cert2, err := x509.ParseCertificate(cert2.Certificate[0])
		require.NoError(t, err)

		// Verify different serial numbers
		assert.NotEqual(t, x509Cert1.SerialNumber, x509Cert2.SerialNumber)

		// Verify different common names
		assert.NotEqual(t, x509Cert1.Subject.CommonName, x509Cert2.Subject.CommonName)
	})

	t.Run("should generate usable TLS certificate", func(t *testing.T) {
		cert, err := generator.GenerateCertificate("tls-test.local")
		require.NoError(t, err)

		// Verify the certificate can be used in TLS config
		tlsConfig := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{*cert},
		}

		assert.NotNil(t, tlsConfig)
		assert.Len(t, tlsConfig.Certificates, 1)
	})

	t.Run("should handle multiple certificate generations", func(t *testing.T) {
		hosts := []string{"host1.local", "host2.local", "host3.local", "host4.local", "host5.local"}

		for _, host := range hosts {
			cert, err := generator.GenerateCertificate(host)
			require.NoError(t, err, "Should generate certificate for %s", host)
			assert.NotNil(t, cert)

			// Verify certificate is for correct host
			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			require.NoError(t, err)
			assert.Equal(t, host, x509Cert.Subject.CommonName)
		}
	})
}
