package server

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

type portListener struct {
	http.Server

	listener net.Listener
}

// bind listens on the target's address, behind TLS when the target asks for it.
func (s *Server) bind(ctx context.Context, target Target) (net.Listener, error) {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", target.Address)
	if err != nil {
		return nil, err
	}

	if target.EnableTLS {
		listener = tls.NewListener(listener, &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: s.manager.getCertificate,
		})
	}

	return listener, nil
}

// Shutdown and Close also close the listener itself: Serve runs on its own
// goroutine and may not have taken the listener over yet.
func (l *portListener) Shutdown(ctx context.Context) error {
	return errors.Join(l.Server.Shutdown(ctx), l.closeListener())
}

func (l *portListener) Close() error {
	return errors.Join(l.Server.Close(), l.closeListener())
}

func (l *portListener) closeListener() error {
	err := l.listener.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}

	return err
}
