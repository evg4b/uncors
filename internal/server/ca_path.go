package server

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const (
	CACertFileName = "ca.crt"
	CAKeyFileName  = "ca.key"
)

func GetCAPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "uncors"), nil
}

func CAExists(fs afero.Fs) bool {
	caDir, err := GetCAPath()
	if err != nil {
		return false
	}

	certPath := filepath.Join(caDir, CACertFileName)
	keyPath := filepath.Join(caDir, CAKeyFileName)

	_, certErr := fs.Stat(certPath)
	_, keyErr := fs.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

func LoadDefaultCA(fs afero.Fs) (*x509.Certificate, *rsa.PrivateKey, error) {
	caDir, err := GetCAPath()
	if err != nil {
		return nil, nil, err
	}

	certPath := filepath.Join(caDir, CACertFileName)
	keyPath := filepath.Join(caDir, CAKeyFileName)

	return LoadCA(fs, certPath, keyPath)
}
