package server

import (
	"context"
	"net"
	"sync"
	"time"
)

const hijackedPollInterval = 10 * time.Millisecond

type hijackedConns struct {
	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
}

func (h *hijackedConns) track(conn net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns == nil {
		h.conns = map[net.Conn]struct{}{}
	}

	h.conns[conn] = struct{}{}

	if h.closed {
		_ = conn.Close()
	}
}

func (h *hijackedConns) release(conn net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.conns, conn)
}

func (h *hijackedConns) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true

	for conn := range h.conns {
		_ = conn.Close()
	}
}

func (h *hijackedConns) active() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.conns)
}

func (h *hijackedConns) wait(ctx context.Context) error {
	ticker := time.NewTicker(hijackedPollInterval)
	defer ticker.Stop()

	for h.active() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}

	return nil
}
