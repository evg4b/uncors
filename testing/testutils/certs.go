// nolint: gomnd
package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
)

type Certs struct {
	ServerTLSConf *tls.Config
	ClientTLSConf *tls.Config
	CertPath      string
	KeyPath       string
}

func WithTmpCerts(fs afero.Fs, action func(t *testing.T, certs *Certs)) func(t *testing.T) {
	if fs == nil {
		fs = afero.NewOsFs()
	}

	return func(t *testing.T) {
		certs := certSetup(t, fs)
		action(t, certs)
	}
}

func certSetup(t *testing.T, fs afero.Fs) *Certs {
	t.Helper()

	now := time.Now()
	currentYear := now.Year()
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(int64(currentYear)),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"CA"},
			Province:      []string{""},
			Locality:      []string{"Fredericton"},
			StreetAddress: []string{"Argyle St."},
			PostalCode:    []string{"E3B 1V1"},
		},
		NotBefore: now,
		NotAfter:  now.AddDate(0, 0, 1),
		IsCA:      true,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	CheckNoError(t, err)

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivateKey.PublicKey, caPrivateKey)
	CheckNoError(t, err)

	// pem encode
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	CheckNoError(t, err)

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	CheckNoError(t, err)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivateKey.PublicKey, caPrivateKey)
	CheckNoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	CheckNoError(t, err)

	tmpDir, err := afero.TempDir(fs, "", "uncors_")
	CheckNoError(t, err)

	certPath := path.Join(tmpDir, "test.cert")
	keyPath := path.Join(tmpDir, "test.key")

	err = afero.WriteFile(fs, certPath, certPEM, os.ModePerm)
	CheckNoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivateKey),
	})
	CheckNoError(t, err)

	err = afero.WriteFile(fs, keyPath, privateKeyPEM, os.ModePerm)
	CheckNoError(t, err)

	serverCert, err := tls.X509KeyPair(certPEM, privateKeyPEM)
	CheckNoError(t, err)

	serverTLSConf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCert},
	}

	certsPool := x509.NewCertPool()
	certsPool.AppendCertsFromPEM(caPEM)
	clientTLSConf := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    certsPool,
	}

	return &Certs{
		ServerTLSConf: serverTLSConf,
		ClientTLSConf: clientTLSConf,
		CertPath:      certPath,
		KeyPath:       keyPath,
	}
}
