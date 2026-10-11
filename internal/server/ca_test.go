package server_test

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCA(t *testing.T) {
	t.Run("should generate CA certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}

		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		assert.FileExists(t, certPath)
		assert.FileExists(t, keyPath)

		assert.Equal(t, filepath.Join(tmpDir, "ca.crt"), certPath)
		assert.Equal(t, filepath.Join(tmpDir, "ca.key"), keyPath)

		keyInfo, err := os.Stat(keyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), keyInfo.Mode().Perm())
	})

	t.Run("should create output directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "subdir", "nested")

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    outputDir,
		}

		_, _, err := server.GenerateCA(config)
		require.NoError(t, err)

		assert.DirExists(t, outputDir)
	})

	t.Run("should generate valid certificate with correct properties", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 730,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}

		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, key, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)

		assert.True(t, cert.IsCA)
		assert.Equal(t, "UNCORS Local Development Root CA", cert.Subject.CommonName)
		assert.Contains(t, cert.Subject.Organization, "UNCORS Development CA")
		assert.Contains(t, cert.Subject.Country, "US")

		expectedDuration := time.Duration(730) * 24 * time.Hour
		actualDuration := cert.NotAfter.Sub(cert.NotBefore)
		tolerance := 5 * time.Minute
		assert.InDelta(t, expectedDuration, actualDuration, float64(tolerance))

		assert.NotEqual(t, 0, cert.KeyUsage&x509.KeyUsageDigitalSignature)
		assert.NotEqual(t, 0, cert.KeyUsage&x509.KeyUsageCertSign)
		assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	})
}

func TestLoadCA(t *testing.T) {
	t.Run("should load valid CA certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, key, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("should return error for non-existent certificate file", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, _, err := server.LoadCA(
			afero.NewOsFs(),
			filepath.Join(tmpDir, "nonexistent.crt"),
			filepath.Join(tmpDir, "nonexistent.key"),
		)
		require.Error(t, err)
	})

	t.Run("should return error for non-existent key file", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, _, err := server.GenerateCA(config)
		require.NoError(t, err)

		_, _, err = server.LoadCA(afero.NewOsFs(), certPath, filepath.Join(tmpDir, "nonexistent.key"))
		require.Error(t, err)
	})

	t.Run("should return error for invalid certificate PEM", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidCertPath := filepath.Join(tmpDir, "invalid.crt")
		keyPath := filepath.Join(tmpDir, "test.key")

		err := os.WriteFile(invalidCertPath, []byte("not a valid PEM"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(keyPath, []byte("not a valid key"), 0o600)
		require.NoError(t, err)

		_, _, err = server.LoadCA(afero.NewOsFs(), invalidCertPath, keyPath)
		require.Error(t, err)
	})
}

func TestCheckExpiration(t *testing.T) {
	t.Run("should detect expiring certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 5,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)

		expiresSoon, timeLeft := server.CheckExpiration(cert, 7*24*time.Hour)
		assert.True(t, expiresSoon)
		assert.Positive(t, timeLeft)
		assert.Less(t, timeLeft, 7*24*time.Hour)
	})

	t.Run("should not detect non-expiring certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)

		expiresSoon, timeLeft := server.CheckExpiration(cert, 7*24*time.Hour)
		assert.False(t, expiresSoon)
		assert.Greater(t, timeLeft, 7*24*time.Hour)
	})

	t.Run("should handle already expired certificate", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 1,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, _, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)

		expiresSoon, timeLeft := server.CheckExpiration(cert, 365*24*time.Hour)
		assert.True(t, expiresSoon)
		assert.Positive(t, timeLeft)
	})
}

func TestLoadCA_ErrorCases(t *testing.T) {
	t.Run("should return error for invalid key PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		err = os.WriteFile(keyPath, []byte("not a valid key PEM"), 0o600)
		require.NoError(t, err)

		_, _, err = server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.Error(t, err)
	})

	t.Run("should return error for invalid certificate in PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		certPath := filepath.Join(tmpDir, "invalid.crt")
		keyPath := filepath.Join(tmpDir, "test.key")

		certFile, err := os.Create(certPath)
		require.NoError(t, err)

		_, err = certFile.WriteString("-----BEGIN CERTIFICATE-----\n")
		require.NoError(t, err)
		_, err = certFile.WriteString("aW52YWxpZCBjZXJ0aWZpY2F0ZSBkYXRh\n")
		require.NoError(t, err)
		_, err = certFile.WriteString("-----END CERTIFICATE-----\n")
		require.NoError(t, err)
		err = certFile.Close()
		require.NoError(t, err)

		keyFile, err := os.Create(keyPath)
		require.NoError(t, err)

		_, err = keyFile.WriteString("key")
		require.NoError(t, err)
		err = keyFile.Close()
		require.NoError(t, err)

		_, _, err = server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse certificate")
	})

	t.Run("should return error for invalid private key in PEM", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		keyFile, err := os.Create(keyPath)
		require.NoError(t, err)

		_, err = keyFile.WriteString("-----BEGIN RSA PRIVATE KEY-----\n")
		require.NoError(t, err)
		_, err = keyFile.WriteString("aW52YWxpZCBrZXkgZGF0YQ==\n")
		require.NoError(t, err)
		_, err = keyFile.WriteString("-----END RSA PRIVATE KEY-----\n")
		require.NoError(t, err)
		err = keyFile.Close()
		require.NoError(t, err)

		_, _, err = server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse private key")
	})
}

func TestGenerateCA_ErrorCases(t *testing.T) {
	t.Run("should use default filesystem when nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}

		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		assert.FileExists(t, certPath)
		assert.FileExists(t, keyPath)
	})
}

func TestLoadCA_UseDefaultFilesystem(t *testing.T) {
	t.Run("should use default filesystem when nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		config := server.CAConfig{
			ValidityDays: 365,
			Fs:           afero.NewOsFs(),
			OutputDir:    tmpDir,
		}
		certPath, keyPath, err := server.GenerateCA(config)
		require.NoError(t, err)

		cert, key, err := server.LoadCA(afero.NewOsFs(), certPath, keyPath)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})
}

func TestCheckCAExpiration(t *testing.T) {
	tests := []struct {
		name     string
		validFor time.Duration
		valid    bool
	}{
		{name: "expired", validFor: -24 * time.Hour},
		{name: "expires in hours", validFor: 12 * time.Hour},
		{name: "expires in 5 days", validFor: 5 * 24 * time.Hour},
		{name: "valid for 10 days", validFor: 10 * 24 * time.Hour, valid: true},
		{name: "valid for a year", validFor: 365 * 24 * time.Hour, valid: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := server.CheckCAExpiration(&x509.Certificate{NotAfter: time.Now().Add(testCase.validFor)})

			if testCase.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
