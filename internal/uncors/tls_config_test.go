package uncors //nolint:testpackage // Testing unexported function

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTLSConfig(t *testing.T) {
	t.Run("should return error when no mappings provided", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mappings := config.Mappings{}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Equal(t, infratls.ErrNoMappingsProvided, err)
	})

	t.Run("should auto-generate certificate when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		mappings := config.Mappings{
			{
				From: "https://localhost:8443",
				To:   hosts.Example.HTTP(),
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)
		assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2

		// Test SNI by requesting certificate for localhost
		cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "localhost"})
		require.NoError(t, err)
		require.NotNil(t, cert)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, "localhost")
	})

	t.Run("should return error when CA does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		mappings := config.Mappings{
			{
				From: "https://localhost:8443",
				To:   hosts.Example.HTTP(),
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load CA certificate")
	})

	t.Run("should generate certificate with correct host from mapping", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		testHost := "example.local"
		mappings := config.Mappings{
			{
				From: "https://" + testHost + ":8443",
				To:   hosts.Example.HTTP(),
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)

		// Test SNI by requesting certificate for the specific host
		cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: testHost})
		require.NoError(t, err)
		require.NotNil(t, cert)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, testHost)
	})

	t.Run("should use SNI to serve different certificates for different hosts on same port", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		require.NoError(t, fs.MkdirAll(fakeHome, 0o755))

		// Generate CA for auto-generated certificates
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		// Create two mappings on the same port but different hosts
		mappings := config.Mappings{
			{
				From: "https://api.local:8443",
				To:   "http://api.example.com",
			},
			{
				From: "https://app.local:8443",
				To:   "http://app.example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)

		// Test SNI for api.local - should auto-generate cert for this host
		apiCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "api.local"})
		require.NoError(t, err)
		require.NotNil(t, apiCert)

		apiX509Cert, err := x509.ParseCertificate(apiCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, apiX509Cert.DNSNames, "api.local")

		// Test SNI for app.local - should auto-generate cert for this host
		appCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "app.local"})
		require.NoError(t, err)
		require.NotNil(t, appCert)

		appX509Cert, err := x509.ParseCertificate(appCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, appX509Cert.DNSNames, "app.local")

		// Verify that each certificate is valid for its respective host
		assert.NotContains(t, apiX509Cert.DNSNames, "app.local", "api cert should not contain app.local")
		assert.NotContains(t, appX509Cert.DNSNames, "api.local", "app cert should not contain api.local")
	})
}
