package server

import (
	"crypto/tls"
	"fmt"
	"net"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

type HostCertManager struct {
	fs          afero.Fs
	certManager *infratls.CertManager
}

func NewHostCertManager(fs afero.Fs) *HostCertManager {
	return &HostCertManager{
		fs:          fs,
		certManager: nil,
	}
}

func (m *HostCertManager) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	err := m.loadCaCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	host, ok := extractServerHost(clientHello)
	if !ok {
		return nil, infratls.ErrNoSNIProvided
	}

	cert, err := m.certManager.GetCertificate(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate for %s: %w", host, err)
	}

	return cert, nil
}

func (m *HostCertManager) loadCaCertificate() error {
	if m.certManager != nil {
		return nil
	}

	caCert, caKey, err := infratls.LoadDefaultCA(afero.NewOsFs())
	if err != nil {
		return fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	err = infratls.CheckCAExpiration(caCert)
	if err != nil {
		return fmt.Errorf("CA certificate validation failed: %w", err)
	}

	m.certManager = infratls.NewCertManager(
		infratls.WithCert(caCert, caKey),
	)

	return nil
}

func extractServerHost(clientHello *tls.ClientHelloInfo) (string, bool) {
	if clientHello == nil {
		return "", false
	}

	if clientHello.ServerName != "" {
		return clientHello.ServerName, true
	}

	if clientHello.Conn == nil || clientHello.Conn.LocalAddr() == nil {
		return "", false
	}

	host, _, err := net.SplitHostPort(clientHello.Conn.LocalAddr().String())
	if err != nil {
		return "", false
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		return "", false
	}

	return host, true
}

func buildTLSConfig(fs afero.Fs) (*tls.Config, error) {
	manager := NewHostCertManager(fs)

	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: manager.getCertificate,
	}, nil
}
