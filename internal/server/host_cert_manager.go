package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

type HostCertManager struct {
	fs        afero.Fs
	publisher events.Publisher
	generator *CertGenerator
	cache     map[string]*tls.Certificate
	mutex     sync.RWMutex

	// Each failure is shown once: a browser retries the handshake many times.
	caReported bool
	reported   map[string]bool
}

func NewHostCertManager(fs afero.Fs, publisher events.Publisher) *HostCertManager {
	if publisher == nil {
		publisher = events.NoopPublisher{}
	}

	return &HostCertManager{
		fs:        fs,
		publisher: publisher,
		cache:     make(map[string]*tls.Certificate),
		reported:  make(map[string]bool),
	}
}

func (m *HostCertManager) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ctx := context.Background()
	if clientHello != nil && clientHello.Context() != nil {
		ctx = clientHello.Context()
	}

	ctx, span := telemetry.Start(ctx, "tls.get_certificate")
	defer span.End()

	err := m.ensureCA(ctx)
	if err != nil {
		err = fmt.Errorf("failed to load CA certificate: %w", err)
		telemetry.Fail(span, err)

		return nil, err
	}

	host, ok := extractServerHost(clientHello)
	if !ok {
		telemetry.Fail(span, ErrNoSNIProvided)

		return nil, ErrNoSNIProvided
	}

	if span.IsRecording() {
		span.SetAttributes(attribute.String("uncors.tls.host", host))
	}

	cert, err := m.certificateForHost(ctx, host)
	if err != nil {
		err = fmt.Errorf("failed to get certificate for %s: %w", host, err)
		telemetry.Fail(span, err)

		return nil, err
	}

	return cert, nil
}

func (m *HostCertManager) ensureCA(ctx context.Context) error {
	m.mutex.RLock()

	if m.generator != nil {
		defer m.mutex.RUnlock()

		return nil
	}

	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.generator != nil {
		return nil
	}

	_, span := telemetry.Start(ctx, "tls.ensure_ca")
	defer span.End()

	err := m.loadCA()
	if err != nil {
		telemetry.Fail(span, err)

		if !m.caReported {
			m.caReported = true
			m.publisher.Publish(events.Notice{
				Level: events.NoticeError,
				Text:  "TLS: " + err.Error(),
			})
		}

		return err
	}

	return nil
}

// Runs under the write lock.
func (m *HostCertManager) loadCA() error {
	caCert, caKey, err := LoadDefaultCA(m.fs)
	if err != nil {
		return fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	err = CheckCAExpiration(caCert)
	if err != nil {
		return fmt.Errorf("CA certificate validation failed: %w", err)
	}

	m.generator = NewCertGenerator(caCert, caKey)

	return nil
}

func (m *HostCertManager) certificateForHost(ctx context.Context, host string) (*tls.Certificate, error) {
	m.mutex.RLock()

	if cert, exists := m.cache[host]; exists {
		defer m.mutex.RUnlock()

		return cert, nil
	}

	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cert, exists := m.cache[host]; exists {
		return cert, nil
	}

	_, span := telemetry.Start(ctx, "tls.generate_certificate")
	defer span.End()

	span.SetAttributes(attribute.String("uncors.tls.host", host))

	cert, err := m.generator.GenerateCertificate(host)
	if err != nil {
		err = fmt.Errorf("failed to generate certificate for %s: %w", host, err)
		telemetry.Fail(span, err)

		if !m.reported[host] {
			m.reported[host] = true
			m.publisher.Publish(events.Notice{Level: events.NoticeError, Text: "TLS: " + err.Error()})
		}

		return nil, err
	}

	m.cache[host] = cert

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
