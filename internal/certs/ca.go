// Package certs manages the local CA that `uncors generate-certs` creates
// and the host certificates signed with it on the first TLS handshake.
package certs

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

const (
	CertFileName = "ca.crt"
	KeyFileName  = "ca.key"

	dirPermissions      = 0o755
	certFilePermissions = 0o644
	keyFilePermissions  = 0o600
	pemBlockCertificate = "CERTIFICATE"
	pemBlockRSAPrivate  = "RSA PRIVATE KEY"
	defaultOrganization = "UNCORS Development CA"
	defaultCommonName   = "UNCORS Local Development Root CA"
	defaultCountry      = "US"

	expirationWarningThreshold = 7 * 24 * time.Hour
	hoursInDay                 = 24
)

func Dir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "uncors"), nil
}

func Paths(dir string) (string, string) {
	return filepath.Join(dir, CertFileName), filepath.Join(dir, KeyFileName)
}

func Exists(fs afero.Fs) bool {
	dir, err := Dir()
	if err != nil {
		return false
	}

	certPath, keyPath := Paths(dir)

	_, certErr := fs.Stat(certPath)
	_, keyErr := fs.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

func Generate(fs afero.Fs, dir string, validityDays int) (string, string, error) {
	notBefore := time.Now()
	template := &x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{defaultOrganization},
			CommonName:   defaultCommonName,
			Country:      []string{defaultCountry},
		},
		NotBefore:             notBefore,
		NotAfter:              notBefore.AddDate(0, 0, validityDays),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	certDER, privateKey, err := issueCA(template)
	if err != nil {
		return "", "", err
	}

	err = fs.MkdirAll(dir, dirPermissions)
	if err != nil {
		return "", "", fmt.Errorf("failed to create output directory: %w", err)
	}

	certPath, keyPath := Paths(dir)

	err = writePEM(fs, certPath, certFilePermissions, pemBlockCertificate, certDER)
	if err != nil {
		return "", "", fmt.Errorf("failed to write certificate: %w", err)
	}

	err = writePEM(fs, keyPath, keyFilePermissions, pemBlockRSAPrivate, x509.MarshalPKCS1PrivateKey(privateKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to write private key: %w", err)
	}

	return certPath, keyPath, nil
}

// The file is created with its final permissions, so a key is never readable by others.
func writePEM(fs afero.Fs, path string, perm os.FileMode, blockType string, data []byte) (err error) {
	file, err := fs.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, file.Close())
	}()

	err = fs.Chmod(path, perm)
	if err != nil {
		return err
	}

	return pem.Encode(file, &pem.Block{Type: blockType, Bytes: data})
}

func Load(fs afero.Fs, certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := afero.ReadFile(fs, certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != pemBlockCertificate {
		return nil, nil, ErrInvalidCertificatePEM
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	keyPEM, err := afero.ReadFile(fs, keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != pemBlockRSAPrivate {
		return nil, nil, ErrInvalidPrivateKeyPEM
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return cert, privateKey, nil
}

func LoadDefault(fs afero.Fs) (*x509.Certificate, *rsa.PrivateKey, error) {
	dir, err := Dir()
	if err != nil {
		return nil, nil, err
	}

	certPath, keyPath := Paths(dir)

	return Load(fs, certPath, keyPath)
}

// CheckExpiration returns ErrCACertExpired for an expired CA. ErrCACertExpiringSoon
// (within a week) is only a warning: the CA still works.
func CheckExpiration(cert *x509.Certificate) error {
	timeLeft := time.Until(cert.NotAfter)

	switch {
	case timeLeft >= expirationWarningThreshold:
		return nil
	case timeLeft < 0:
		return ErrCACertExpired
	case timeLeft < hoursInDay*time.Hour:
		return fmt.Errorf("CA certificate expires in less than %d hours! %w", int(timeLeft.Hours()), ErrCACertExpiringSoon)
	default:
		return fmt.Errorf("CA certificate expires in %d days! %w", int(timeLeft.Hours()/hoursInDay), ErrCACertExpiringSoon)
	}
}
