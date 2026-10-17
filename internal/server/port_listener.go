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
	hijacked *hijackedConns
}

func (s *Server) bind(ctx context.Context, target Target) (net.Listener, error) {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", target.Address)
	if err != nil {
		return nil, err
	}

	if target.EnableTLS {
		listener = tls.NewListener(listener, &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: s.getCertificate,
		})
	}

	return listener, nil
}

// Shutdown and Close also close the listener itself: Serve runs on its own
// goroutine and may not have taken the listener over yet.
func (l *portListener) Shutdown(ctx context.Context) error {
	l.hijacked.closeAll()

	return errors.Join(l.Server.Shutdown(ctx), l.closeListener(), l.hijacked.wait(ctx))
}

func (l *portListener) Close() error {
	l.hijacked.closeAll()

	return errors.Join(l.Server.Close(), l.closeListener())
}

func (l *portListener) closeListener() error {
	err := l.listener.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}

	return err
}
