package uncors //nolint:testpackage // Testing unexported function

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCertPath = "/test/cert.crt"
	testKeyPath  = "/test/key.key"
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

	t.Run("should load custom certificate when cert and key files provided", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := testCertPath
		keyPath := testKeyPath

		tmpDir := t.TempDir()

		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    tmpDir,
		}
		generatedCertPath, generatedKeyPath, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		certData, err := os.ReadFile(generatedCertPath)
		require.NoError(t, err)
		keyData, err := os.ReadFile(generatedKeyPath)
		require.NoError(t, err)

		require.NoError(t, afero.WriteFile(fs, certPath, certData, 0o600))
		require.NoError(t, afero.WriteFile(fs, keyPath, keyData, 0o600))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  keyPath,
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.NotNil(t, tlsConfig.GetCertificate)
		assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2
	})

	t.Run("should return error when cert file does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: "/nonexistent/cert.crt",
				KeyFile:  "/nonexistent/key.key",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to read certificate file")
	})

	t.Run("should return error when key file does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := testCertPath
		require.NoError(t, afero.WriteFile(fs, certPath, []byte("cert data"), 0o600))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  "/nonexistent/key.key",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to read key file")
	})

	t.Run("should return error for invalid certificate data", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := testCertPath
		keyPath := testKeyPath

		require.NoError(t, afero.WriteFile(fs, certPath, []byte("invalid cert"), 0o600))
		require.NoError(t, afero.WriteFile(fs, keyPath, []byte("invalid key"), 0o600))

		mappings := config.Mappings{
			{
				From:     "https://localhost:8443",
				To:       "http://example.com",
				CertFile: certPath,
				KeyFile:  keyPath,
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load certificate")
	})

	t.Run("should auto-generate certificate when no custom cert provided and CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

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
				To:   "http://example.com",
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

	t.Run("should return error when CA does not exist and no custom cert provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		mappings := config.Mappings{
			{
				From: "https://localhost:8443",
				To:   "http://example.com",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to load CA certificate")
	})

	t.Run("should return error when mapping has invalid host with custom cert", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		mappings := config.Mappings{
			{
				From:     "https://[invalid-host",
				To:       "http://example.com",
				CertFile: "/cert.crt",
				KeyFile:  "/key.key",
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.Error(t, err)
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "failed to parse mapping host")
	})

	t.Run("should generate certificate with correct host from mapping", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

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
				To:   "http://example.com",
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
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

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

	t.Run("should mix custom and auto-generated certificates on same port", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		// Generate CA for auto-generated certificates
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		// Create custom certificate for one host
		customCertDir := filepath.Join(tmpDir, "custom")
		require.NoError(t, os.MkdirAll(customCertDir, 0o755))
		customCertConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    customCertDir,
		}
		customCertPath, customKeyPath, err := infratls.GenerateCA(customCertConfig)
		require.NoError(t, err)

		mappings := config.Mappings{
			{
				From:     "https://api.local:8443",
				To:       "http://api.example.com",
				CertFile: customCertPath,
				KeyFile:  customKeyPath,
			},
			{
				From: "https://app.local:8443",
				To:   "http://app.example.com",
				// No cert/key - should use auto-generated
			},
		}

		tlsConfig, err := buildTLSConfig(fs, mappings)

		require.NoError(t, err)
		require.NotNil(t, tlsConfig)

		// Test custom certificate
		apiCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "api.local"})
		require.NoError(t, err)
		require.NotNil(t, apiCert)

		// Test auto-generated certificate
		appCert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "app.local"})
		require.NoError(t, err)
		require.NotNil(t, appCert)

		// Verify app.local has correct DNS name
		x509Cert, err := x509.ParseCertificate(appCert.Certificate[0])
		require.NoError(t, err)
		assert.Contains(t, x509Cert.DNSNames, "app.local")
	})
}
