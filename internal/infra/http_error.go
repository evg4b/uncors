package infra

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/headers"
)

var ErrHostNotMapped = errors.New("host not mapped")

const errorHeader = `
███████  ██████   ██████      ███████ ██████  ██████   ██████  ██████  
██      ██  ████ ██  ████     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████ ██ ██ ██ ██ ██ ██     █████   ██████  ██████  ██    ██ ██████  
     ██ ████  ██ ████  ██     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████  ██████   ██████      ███████ ██   ██ ██   ██  ██████  ██   ██ `

func HTTPError(writer contracts.ResponseWriter, request *http.Request, err error) {
	if request.Context().Err() != nil || ResponseStarted(writer) {
		panic(http.ErrAbortHandler)
	}

	header := writer.Header()
	header.Set(headers.ContentType, "text/plain; charset=utf-8")
	header.Set(headers.ContentEncoding, "identity")
	header.Set(headers.CacheControl, "no-cache, no-store, max-age=0, must-revalidate")
	header.Set(headers.Pragma, "no-cache")
	header.Set(headers.XContentTypeOptions, "nosniff")

	header.Del(headers.SetCookie)
	header.Del(headers.ContentLength)

	WriteRequestCorsHeaders(header, request)

	writer.WriteHeader(errorStatus(err))

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, errorHeader)
	fmt.Fprintln(writer)
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "An error occurred: %s\n", err)
}

func ResponseStarted(writer contracts.ResponseWriter) bool {
	if writer.StatusCode() != 0 {
		return true
	}

	counter, ok := writer.(interface{ Written() int64 })

	return ok && counter.Written() > 0
}

func errorStatus(err error) int {
	if errors.Is(err, ErrHostNotMapped) {
		return http.StatusNotFound
	}

	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return http.StatusGatewayTimeout
	}

	if isUpstreamError(err) {
		return http.StatusBadGateway
	}

	return http.StatusInternalServerError
}

func isUpstreamError(err error) bool {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		return urlErr.Op != "parse"
	}

	var opErr *net.OpError

	return errors.As(err, &opErr)
}
