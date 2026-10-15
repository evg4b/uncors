package certs

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTmpDir = "/tmp/test"

func TestIssueHostCertificate(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmpDir := testTmpDir
	certPath, keyPath, err := Generate(fs, tmpDir, 365)
	require.NoError(t, err)

	caCert, caKey, err := Load(fs, certPath, keyPath)
	require.NoError(t, err)

	t.Run("should generate certificate for localhost", func(t *testing.T) {
		cert, err := issueHostCertificate(caCert, caKey, "localhost")
		require.NoError(t, err)
		assert.NotNil(t, cert)

		assert.NotNil(t, cert.PrivateKey)
		assert.NotEmpty(t, cert.Certificate)
	})

	t.Run("should generate certificate for custom domain", func(t *testing.T) {
		cert, err := issueHostCertificate(caCert, caKey, "example.local")
		require.NoError(t, err)
		assert.NotNil(t, cert)
	})

	t.Run("should generate valid certificate with correct properties", func(t *testing.T) {
		host := "test.local"
		cert, err := issueHostCertificate(caCert, caKey, host)
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		assert.Equal(t, host, x509Cert.Subject.CommonName)
		assert.Contains(t, x509Cert.DNSNames, host)
		assert.Contains(t, x509Cert.Subject.Organization, "UNCORS Development CA")

		assert.NotEqual(t, 0, x509Cert.KeyUsage&x509.KeyUsageDigitalSignature)
		assert.NotEqual(t, 0, x509Cert.KeyUsage&x509.KeyUsageKeyEncipherment)
		assert.Contains(t, x509Cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

		assert.False(t, x509Cert.IsCA)

		expectedDuration := 365 * 24 * time.Hour
		actualDuration := x509Cert.NotAfter.Sub(x509Cert.NotBefore)
		tolerance := 5 * time.Minute
		assert.InDelta(t, expectedDuration, actualDuration, float64(tolerance))
	})

	t.Run("should generate certificate signed by CA", func(t *testing.T) {
		cert, err := issueHostCertificate(caCert, caKey, "verify.local")
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

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
		cert1, err := issueHostCertificate(caCert, caKey, "host1.local")
		require.NoError(t, err)

		cert2, err := issueHostCertificate(caCert, caKey, "host2.local")
		require.NoError(t, err)

		x509Cert1, err := x509.ParseCertificate(cert1.Certificate[0])
		require.NoError(t, err)
		x509Cert2, err := x509.ParseCertificate(cert2.Certificate[0])
		require.NoError(t, err)

		assert.NotEqual(t, x509Cert1.SerialNumber, x509Cert2.SerialNumber)

		assert.NotEqual(t, x509Cert1.Subject.CommonName, x509Cert2.Subject.CommonName)
	})

	t.Run("should generate usable TLS certificate", func(t *testing.T) {
		cert, err := issueHostCertificate(caCert, caKey, "tls-test.local")
		require.NoError(t, err)

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
			cert, err := issueHostCertificate(caCert, caKey, host)
			require.NoError(t, err, "Should generate certificate for %s", host)
			assert.NotNil(t, cert)

			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			require.NoError(t, err)
			assert.Equal(t, host, x509Cert.Subject.CommonName)
		}
	})
}
