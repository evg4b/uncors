package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

const (
	certValidityDays = 365
)

type CertGenerator struct {
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
}

func NewCertGenerator(caCert *x509.Certificate, caKey *rsa.PrivateKey) *CertGenerator {
	return &CertGenerator{
		caCert: caCert,
		caKey:  caKey,
	}
}

func (g *CertGenerator) GenerateCertificate(host string) (*tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, certValidityDays)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{defaultOrganization},
			CommonName:   host,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, g.caCert, &privateKey.PublicKey, g.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  pemBlockCertificate,
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  pemBlockRSAPrivate,
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate: %w", err)
	}

	return &tlsCert, nil
}
