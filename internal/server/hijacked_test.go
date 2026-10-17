package server_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hijackWait = 5 * time.Second

type syncPublisher struct {
	mu        sync.Mutex
	completed []events.RequestCompleted
}

func (p *syncPublisher) Publish(event events.Event) {
	if typed, ok := event.(events.RequestCompleted); ok {
		p.mu.Lock()
		p.completed = append(p.completed, typed)
		p.mu.Unlock()
	}
}

func (p *syncPublisher) codes() []int {
	p.mu.Lock()
	defer p.mu.Unlock()

	codes := make([]int, 0, len(p.completed))
	for _, event := range p.completed {
		codes = append(codes, event.Data.Code)
	}

	return codes
}

type hijackingTarget struct {
	hijacked chan struct{}
	finished chan struct{}
	release  chan struct{}
}

func newHijackingTarget(ignoreClose bool) *hijackingTarget {
	target := &hijackingTarget{
		hijacked: make(chan struct{}, 1),
		finished: make(chan struct{}, 1),
		release:  make(chan struct{}),
	}

	if !ignoreClose {
		close(target.release)
	}

	return target
}

func (h *hijackingTarget) handler(t *testing.T) http.Handler {
	t.Helper()

	return infra.CastToHTTPHandler(infra.HandlerFunc(func(writer contracts.ResponseWriter, _ *contracts.Request) error {
		writer.Header().Set("Connection", "Upgrade")
		writer.Header().Set("Upgrade", "test")
		writer.WriteHeader(http.StatusSwitchingProtocols)

		conn, buffered, err := http.NewResponseController(writer).Hijack()
		if err != nil {
			return err
		}

		h.hijacked <- struct{}{}

		_, _ = io.Copy(io.Discard, buffered)

		<-h.release

		_ = conn.Close()

		h.finished <- struct{}{}

		return nil
	}))
}

func startHijacking(t *testing.T, target *hijackingTarget, publisher events.Publisher) (*server.Server, string) {
	t.Helper()

	address := hosts.Loopback.Port(testutils.GetFreePorts(t, 1)[0]).String()
	instance := server.New(certs.NewManager(afero.NewMemMapFs(), publisher).GetCertificate, publisher)
	require.NoError(t, instance.Start(t.Context(), []server.Target{{
		Address: address,
		Handler: target.handler(t),
	}}))

	t.Cleanup(func() { _ = instance.Close() })

	return instance, address
}

func upgrade(t *testing.T, address string, target *hijackingTarget) net.Conn {
	t.Helper()

	var dialer net.Dialer

	conn, err := dialer.DialContext(t.Context(), "tcp", address)
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	_, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: Upgrade\r\nUpgrade: test\r\n\r\n")
	require.NoError(t, err)

	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)

	waitFor(t, target.hijacked, "hijack")

	return conn
}

func waitFor(t *testing.T, signal <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(hijackWait):
		t.Fatalf("no %s within %s", what, hijackWait)
	}
}

func assertClosedByServer(t *testing.T, conn net.Conn) {
	t.Helper()

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(hijackWait)))

	_, err := conn.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF)
}

func TestServerHijackedConnections(t *testing.T) {
	t.Run("shutdown closes them and waits for their handlers", func(t *testing.T) {
		publisher := &syncPublisher{}
		target := newHijackingTarget(false)
		instance, address := startHijacking(t, target, publisher)
		conn := upgrade(t, address, target)

		require.NoError(t, instance.Shutdown(t.Context()))

		select {
		case <-target.finished:
		default:
			t.Fatal("shutdown returned before the hijacking handler finished")
		}

		assert.Equal(t, []int{http.StatusSwitchingProtocols}, publisher.codes())
		assertClosedByServer(t, conn)
	})

	t.Run("close closes them", func(t *testing.T) {
		target := newHijackingTarget(false)
		instance, address := startHijacking(t, target, events.NoopPublisher{})
		conn := upgrade(t, address, target)

		require.NoError(t, instance.Close())

		waitFor(t, target.finished, "finished handler")
		assertClosedByServer(t, conn)
	})

	t.Run("restart closes them", func(t *testing.T) {
		target := newHijackingTarget(false)
		instance, address := startHijacking(t, target, events.NoopPublisher{})
		conn := upgrade(t, address, target)

		require.NoError(t, instance.Restart(t.Context(), []server.Target{{
			Address: address,
			Handler: target.handler(t),
		}}))

		waitFor(t, target.finished, "finished handler")
		assertClosedByServer(t, conn)
	})

	t.Run("shutdown reports handlers that outlive it", func(t *testing.T) {
		target := newHijackingTarget(true)
		instance, address := startHijacking(t, target, events.NoopPublisher{})
		upgrade(t, address, target)

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		require.ErrorIs(t, instance.Shutdown(ctx), context.DeadlineExceeded)

		close(target.release)
		waitFor(t, target.finished, "finished handler")
	})
}
