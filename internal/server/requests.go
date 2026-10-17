package server

import (
	"errors"
	"net"
	"sync"
)

type requests struct {
	mu      sync.Mutex
	active  int
	conns   map[net.Conn]int
	retired bool
	idle    chan struct{}

	abortOnce sync.Once
	aborted   chan struct{}
}

func newRequests() *requests {
	return &requests{
		conns:   map[net.Conn]int{},
		idle:    make(chan struct{}),
		aborted: make(chan struct{}),
	}
}

func (r *requests) acquire(conn net.Conn) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.retired {
		return false
	}

	r.active++

	if conn != nil {
		r.conns[conn]++
	}

	return true
}

func (r *requests) release(conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.active--

	if conn != nil {
		r.conns[conn]--
		if r.conns[conn] <= 0 {
			delete(r.conns, conn)
		}
	}

	if r.retired && r.active == 0 {
		close(r.idle)
	}
}

func (r *requests) retire() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.retired {
		r.retired = true
		if r.active == 0 {
			close(r.idle)
		}
	}

	return r.idle
}

func (r *requests) abort() {
	r.abortOnce.Do(func() {
		close(r.aborted)
	})
}

func (r *requests) closeConns() error {
	r.mu.Lock()
	conns := make([]net.Conn, 0, len(r.conns))

	for conn := range r.conns {
		conns = append(conns, conn)
	}
	r.mu.Unlock()

	var errs []error

	for _, conn := range conns {
		err := conn.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
