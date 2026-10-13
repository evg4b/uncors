package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"time"
)

const (
	keySize          = 2048
	serialNumberBits = 128
	certValidityDays = 365
)

// issue signs template with a new key and serial number. A nil parent
// makes the certificate self-signed.
func issue(template, parent *x509.Certificate, parentKey *rsa.PrivateKey) ([]byte, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	template.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	if parent == nil {
		parent, parentKey = template, privateKey
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, &privateKey.PublicKey, parentKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return certDER, privateKey, nil
}

// issueHostCertificate signs a certificate for host, a name or an IP
// address, with the CA.
func issueHostCertificate(caCert *x509.Certificate, caKey *rsa.PrivateKey, host string) (*tls.Certificate, error) {
	notBefore := time.Now()
	template := &x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{defaultOrganization},
			CommonName:   host,
		},
		NotBefore:             notBefore,
		NotAfter:              notBefore.AddDate(0, 0, certValidityDays),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, privateKey, err := issue(template, caCert, caKey)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: privateKey}, nil
}
