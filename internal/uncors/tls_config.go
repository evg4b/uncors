package uncors

import (
	"crypto/tls"
	"fmt"
	"net"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
)

type hostCertManager struct {
	certManager *infratls.CertManager // for auto-generated certificates
}

func newHostCertManager(fs afero.Fs) (*hostCertManager, error) {
	// Load CA for auto-generation
	caCert, caKey, err := infratls.LoadDefaultCA(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	err = infratls.CheckCAExpiration(caCert)
	if err != nil {
		return nil, fmt.Errorf("CA certificate validation failed: %w", err)
	}

	return &hostCertManager{
		certManager: infratls.NewCertManager(
			infratls.WithCert(caCert, caKey),
		),
	}, nil
}

func (m *hostCertManager) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
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
	manager, err := newHostCertManager(fs)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: manager.getCertificate,
	}, nil
}
