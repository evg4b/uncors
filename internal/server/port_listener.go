package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
)

type portListener struct {
	http.Server

	target   Target
	listener net.Listener
	route    atomic.Pointer[route]
	cancel   context.CancelFunc
}

func (s *Server) bind(ctx context.Context, target Target) (net.Listener, error) {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", target.Address)
	if err != nil {
		_, port, _ := net.SplitHostPort(target.Address)

		return nil, fmt.Errorf("cannot listen on port %s: %w", port, err)
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
	l.cancel()

	return l.drain(ctx)
}

func (l *portListener) Close() error {
	defer l.cancel()

	return errors.Join(ignoreClosed(l.Server.Close()), l.closeListener())
}

func (l *portListener) drain(ctx context.Context) error {
	return errors.Join(ignoreClosed(l.Server.Shutdown(ctx)), l.closeListener())
}

func (l *portListener) closeListener() error {
	return ignoreClosed(l.listener.Close())
}

func ignoreClosed(err error) error {
	if errors.Is(err, net.ErrClosed) {
		return nil
	}

	return err
}
