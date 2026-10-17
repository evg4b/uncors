package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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

func issueCA(template *x509.Certificate) ([]byte, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	certDER, err := sign(template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	return certDER, privateKey, nil
}

func sign(template, parent *x509.Certificate, publicKey, parentKey any) ([]byte, error) {
	var err error

	template.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return certDER, nil
}

func issueHostCertificate(caCert *x509.Certificate, caKey *rsa.PrivateKey, host string) (*tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()

	notAfter := notBefore.AddDate(0, 0, certValidityDays)
	if notAfter.After(caCert.NotAfter) {
		notAfter = caCert.NotAfter
	}

	template := &x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{defaultOrganization},
			CommonName:   host,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, err := sign(template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: privateKey}, nil
}
