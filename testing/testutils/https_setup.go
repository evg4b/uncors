package testutils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"path/filepath"
	"testing"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

const (
	defaultDirPermissions = 0o755
	defaultCAValidityDays = 365
)

// SetupHTTPSTest sets up CA for HTTPS tests and returns HTTP client with proper TLS config.
func SetupHTTPSTest(t *testing.T, fs afero.Fs) *http.Client {
	t.Helper()

	tmpDir := t.TempDir()
	fakeHome := filepath.Join(tmpDir, "home")
	CheckNoError(t, fs.MkdirAll(fakeHome, defaultDirPermissions))
	t.Setenv("HOME", fakeHome)

	// Generate CA using uncors
	caDir := filepath.Join(fakeHome, ".config", "uncors")
	caConfig := infratls.CAConfig{
		ValidityDays: defaultCAValidityDays,
		OutputDir:    caDir,
		Fs:           fs,
	}
	certPath, _, err := infratls.GenerateCA(caConfig)
	CheckNoError(t, err)

	// Load CA certificate for client
	caCertData, err := afero.ReadFile(fs, certPath)
	CheckNoError(t, err)
	CheckNoError(t, err)

	// Setup client TLS config to trust the CA
	certsPool := x509.NewCertPool()
	certsPool.AppendCertsFromPEM(caCertData)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    certsPool,
			},
		},
	}
}
