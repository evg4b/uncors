package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/otel/attribute"
)

var errUpstreamNotWritable = errors.New("upstream switched protocols without a writable connection")

func isProtocolSwitch(request *http.Request, response *http.Response) bool {
	return response.StatusCode == http.StatusSwitchingProtocols && infra.IsUpgrade(request.Header)
}

func (h *Handler) switchProtocols(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
	req *http.Request,
) error {
	upstream, ok := original.Body.(io.ReadWriteCloser)
	if !ok {
		helpers.CloseSafe(original.Body)

		return errUpstreamNotWritable
	}

	copyCookiesToSource(original, replacer, target)

	copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: replacer.ReplaceSoft,
	})

	infra.WriteRequestCorsHeaders(target.Header(), req)

	target.WriteHeader(http.StatusSwitchingProtocols)

	client, buffered, err := http.NewResponseController(target).Hijack()
	if err != nil {
		_ = upstream.Close()

		return fmt.Errorf("failed to take over the client connection: %w", err)
	}

	tunnel(req.Context(), client, buffered.Reader, upstream)

	return nil
}

func tunnel(ctx context.Context, client net.Conn, clientReader io.Reader, upstream io.ReadWriteCloser) {
	ctx, span := telemetry.Start(ctx, "proxy.tunnel")
	defer span.End()

	closeBoth := func() {
		_ = client.Close()
		_ = upstream.Close()
	}

	stop := context.AfterFunc(ctx, closeBoth)
	defer stop()

	sent := make(chan int64, 1)

	go func() {
		written := pipe(upstream, clientReader)

		closeBoth()

		sent <- written
	}()

	received := pipe(client, upstream)

	closeBoth()

	toUpstream := <-sent

	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("uncors.proxy.tunnel.sent_bytes", toUpstream),
			attribute.Int64("uncors.proxy.tunnel.received_bytes", received),
			attribute.Bool("uncors.proxy.tunnel.cancelled", ctx.Err() != nil),
		)
	}
}

func pipe(target io.Writer, source io.Reader) int64 {
	buffer, _ := copyBuffers.Get().(*[]byte)
	defer copyBuffers.Put(buffer)

	written, _ := io.CopyBuffer(target, source, *buffer)

	return written
}
