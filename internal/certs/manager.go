package certs

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Manager loads the CA on the first TLS handshake and signs a certificate
// for every host name once.
type Manager struct {
	fs        afero.Fs
	publisher events.Publisher

	// Guards everything below; ensureCA, loadCA and certificateForHost run under it.
	mu     sync.Mutex
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
	cache  map[string]*tls.Certificate

	// Each failure is shown once: a browser retries the handshake many times.
	caReported bool
	reported   map[string]bool
}

func NewManager(fs afero.Fs, publisher events.Publisher) *Manager {
	return &Manager{
		fs:        fs,
		publisher: publisher,
		cache:     make(map[string]*tls.Certificate),
		reported:  make(map[string]bool),
	}
}

func (m *Manager) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ctx := clientHello.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, span := telemetry.Start(ctx, "tls.get_certificate")
	defer span.End()

	if span.IsRecording() {
		annotateClientHello(span, clientHello)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *Manager) ensureCA(ctx context.Context) error {
	if m.caCert != nil {
		return nil
	}

	ctx, span := telemetry.Start(ctx, "tls.ensure_ca")
	defer span.End()

	err := m.loadCA(ctx)
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

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.tls.ca.subject", m.caCert.Subject.String()),
			attribute.String("uncors.tls.ca.not_after", m.caCert.NotAfter.UTC().Format(time.RFC3339)),
		)
	}

	return nil
}

func (m *Manager) loadCA(ctx context.Context) error {
	caCert, caKey, err := LoadDefault(m.fs)
	if err != nil {
		return fmt.Errorf("failed to load CA certificate for auto-generation: %w", err)
	}

	err = CheckExpiration(caCert)
	if errors.Is(err, ErrCACertExpiringSoon) {
		telemetry.Warn(ctx, "CA certificate expires soon", telemetry.Err(err))
		m.publisher.Publish(events.Notice{Level: events.NoticeWarning, Text: "TLS: " + err.Error()})
	} else if err != nil {
		return fmt.Errorf("CA certificate validation failed: %w", err)
	}

	m.caCert, m.caKey = caCert, caKey

	return nil
}

func (m *Manager) certificateForHost(ctx context.Context, host string) (*tls.Certificate, error) {
	if cert, exists := m.cache[host]; exists {
		return cert, nil
	}

	_, span := telemetry.Start(ctx, "tls.generate_certificate")
	defer span.End()

	span.SetAttributes(attribute.String("uncors.tls.host", host))

	cert, err := issueHostCertificate(m.caCert, m.caKey, host)
	telemetry.TLSCertificate(ctx, err)

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

func annotateClientHello(span trace.Span, clientHello *tls.ClientHelloInfo) {
	attrs := []attribute.KeyValue{
		attribute.String("tls.client.server_name", clientHello.ServerName),
	}

	if len(clientHello.SupportedProtos) > 0 {
		attrs = append(attrs, attribute.StringSlice("tls.client.supported_protocols", clientHello.SupportedProtos))
	}

	if clientHello.Conn != nil && clientHello.Conn.RemoteAddr() != nil {
		attrs = append(attrs, semconv.ClientAddress(clientHello.Conn.RemoteAddr().String()))
	}

	span.SetAttributes(attrs...)
}

// Without SNI the local address the client connected to is used.
func extractServerHost(clientHello *tls.ClientHelloInfo) (string, bool) {
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
