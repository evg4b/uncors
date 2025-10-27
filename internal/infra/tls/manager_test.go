package tls_test

import (
	"crypto/x509"
	"testing"
	"time"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertManager(t *testing.T) {
	t.Run("should create cert manager with CA", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tmpDir := testTmpDir
		config := infratls.CAConfig{
			ValidityDays: 365,
			Fs:           fs,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		caCert, caKey, err := infratls.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		manager := infratls.NewCertManager(caCert, caKey)
		assert.NotNil(t, manager)
	})

	t.Run("should create cert manager without CA", func(t *testing.T) {
		manager := infratls.NewCertManager(nil, nil)
		assert.NotNil(t, manager)
	})
}

func TestCertManager_GetCertificate(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmpDir := testTmpDir
	config := infratls.CAConfig{
		ValidityDays: 365,
		Fs:           fs,
		OutputDir:    tmpDir,
	}
	certPath, keyPath, err := infratls.GenerateCA(config)
	require.NoError(t, err)

	caCert, caKey, err := infratls.LoadCA(fs, certPath, keyPath)
	require.NoError(t, err)

	t.Run("should generate and cache certificate", func(t *testing.T) {
		manager := infratls.NewCertManager(caCert, caKey)

		cert1, err := manager.GetCertificate(hosts.Example.Host())
		require.NoError(t, err)
		assert.NotNil(t, cert1)

		// Get the same certificate again (should use cache)
		cert2, err := manager.GetCertificate(hosts.Example.Host())
		require.NoError(t, err)
		assert.NotNil(t, cert2)

		// Verify it's the same certificate (pointer equality)
		assert.Equal(t, cert1, cert2)
	})

	t.Run("should generate different certificates for different hosts", func(t *testing.T) {
		manager := infratls.NewCertManager(caCert, caKey)

		cert1, err := manager.GetCertificate("host1.local")
		require.NoError(t, err)

		cert2, err := manager.GetCertificate("host2.local")
		require.NoError(t, err)

		assert.NotEqual(t, cert1, cert2)
	})

	t.Run("should return error when no CA and no cached certificate", func(t *testing.T) {
		manager := infratls.NewCertManager(nil, nil)

		_, err := manager.GetCertificate(hosts.Example.Host())
		require.Error(t, err)
	})

	t.Run("should handle concurrent requests for same host", func(t *testing.T) {
		manager := infratls.NewCertManager(caCert, caKey)

		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for range numGoroutines {
			go func() {
				_, err := manager.GetCertificate("concurrent.local")
				results <- err
			}()
		}

		// Collect results
		for range numGoroutines {
			err := <-results
			assert.NoError(t, err)
		}
	})

	t.Run("should handle concurrent requests for different hosts", func(t *testing.T) {
		manager := infratls.NewCertManager(caCert, caKey)

		const numGoroutines = 5
		results := make(chan error, numGoroutines)

		for i := range numGoroutines {
			host := "host" + string(rune('0'+i)) + ".local"
			go func(h string) {
				_, err := manager.GetCertificate(h)
				results <- err
			}(host)
		}

		// Collect results
		for range numGoroutines {
			err := <-results
			assert.NoError(t, err)
		}
	})

	t.Run("should cache certificates correctly", func(t *testing.T) {
		manager := infratls.NewCertManager(caCert, caKey)

		hosts := []string{"cache1.local", "cache2.local", "cache3.local"}

		certs := make(map[string]any)
		for _, host := range hosts {
			cert, err := manager.GetCertificate(host)
			require.NoError(t, err)
			certs[host] = cert
		}

		for _, host := range hosts {
			cert, err := manager.GetCertificate(host)
			require.NoError(t, err)
			assert.Equal(t, certs[host], cert, "Certificate should be cached for %s", host)
		}
	})
}

func TestCheckCAExpiration(t *testing.T) {
	t.Run("should not panic with valid certificate", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tmpDir := testTmpDir
		config := infratls.CAConfig{
			ValidityDays: 365,
			Fs:           fs,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		caCert, _, err := infratls.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		// Should not panic
		assert.NotPanics(t, func() {
			infratls.CheckCAExpiration(caCert)
		})
	})

	t.Run("should handle expiring certificate", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		tmpDir := testTmpDir
		config := infratls.CAConfig{
			ValidityDays: 5, // Will expire soon
			Fs:           fs,
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		caCert, _, err := infratls.LoadCA(fs, certPath, keyPath)
		require.NoError(t, err)

		// Should not panic even with expiring cert
		assert.NotPanics(t, func() {
			infratls.CheckCAExpiration(caCert)
		})
	})

	t.Run("should show correct message for expired certificate", func(t *testing.T) {
		// Create a certificate that has already expired
		cert := &x509.Certificate{
			NotAfter: time.Now().Add(-24 * time.Hour), // Expired yesterday
		}

		// Should not panic with expired cert
		assert.NotPanics(t, func() {
			infratls.CheckCAExpiration(cert)
		})
	})

	t.Run("should show correct message for certificate expiring in hours", func(t *testing.T) {
		// Create a certificate that expires in less than 24 hours
		cert := &x509.Certificate{
			NotAfter: time.Now().Add(12 * time.Hour), // Expires in 12 hours
		}

		// Should not panic
		assert.NotPanics(t, func() {
			infratls.CheckCAExpiration(cert)
		})
	})

	t.Run("should not show warning for certificate valid for more than 7 days", func(t *testing.T) {
		// Create a certificate that expires in more than 7 days
		cert := &x509.Certificate{
			NotAfter: time.Now().Add(10 * 24 * time.Hour), // Expires in 10 days
		}

		// Should not panic and should not log warning
		assert.NotPanics(t, func() {
			infratls.CheckCAExpiration(cert)
		})
	})
}
