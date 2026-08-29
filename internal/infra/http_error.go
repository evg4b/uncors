package infra

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"syscall"

	"github.com/go-http-utils/headers"
)

// statusClientClosedRequest is the conventional (non-standard) code for "the
// client went away before we answered". Nothing is written for it; there is
// nobody left to read it.
const statusClientClosedRequest = 499

const errorHeader = `
███████  ██████   ██████      ███████ ██████  ██████   ██████  ██████  
██      ██  ████ ██  ████     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████ ██ ██ ██ ██ ██ ██     █████   ██████  ██████  ██    ██ ██████  
     ██ ████  ██ ████  ██     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████  ██████   ██████      ███████ ██   ██ ██   ██  ██████  ██   ██ `

// HTTPStatusError carries the status and the user-facing message a failure
// should be reported with. Handlers that know what went wrong say so, instead of
// leaving every failure to be reported as an indistinguishable 500.
type HTTPStatusError struct {
	Code int
	Msg  string
	Err  error
}

func NewHTTPStatusError(code int, msg string, err error) *HTTPStatusError {
	return &HTTPStatusError{Code: code, Msg: msg, Err: err}
}

func (e *HTTPStatusError) Error() string {
	if e.Err == nil {
		return e.Msg
	}

	return e.Msg + ": " + e.Err.Error()
}

func (e *HTTPStatusError) Unwrap() error {
	return e.Err
}

// StatusFor maps an error onto the status and message the client should see.
func StatusFor(err error) (int, string) {
	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code, statusErr.Msg
	}

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, http.ErrAbortHandler):
		return statusClientClosedRequest, ""
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "the request to the original source timed out"
	case isUpstreamUnreachable(err):
		return http.StatusBadGateway, "the original source is unreachable"
	default:
		return http.StatusInternalServerError, "uncors could not handle this request"
	}
}

// HTTPError reports a failed request: the detail goes to the log, where a
// developer looks, and the client gets the status and a one-line summary.
func HTTPError(writer http.ResponseWriter, request *http.Request, err error) {
	code, message := StatusFor(err)

	slog.Error("request failed",
		"method", requestMethod(request),
		"url", requestURL(request),
		"status", code,
		"err", err)

	if code == statusClientClosedRequest {
		return
	}

	writeErrorPage(writer, request, code, message, err)
}

func writeErrorPage(writer http.ResponseWriter, request *http.Request, code int, message string, err error) {
	header := writer.Header()
	header.Set(headers.ContentType, "text/plain; charset=utf-8")
	header.Set(headers.ContentEncoding, "identity")
	header.Set(headers.CacheControl, "no-cache, no-store, max-age=0, must-revalidate")
	header.Set(headers.Pragma, "no-cache")
	header.Set(headers.XContentTypeOptions, "nosniff")

	header.Del(headers.SetCookie)

	writer.WriteHeader(code)

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, errorHeader)
	fmt.Fprintln(writer)
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Error %d: %s\n", code, message)
	fmt.Fprintln(writer)
	// Echoing the request back is the point of the page. It is served as
	// text/plain with nosniff, so a browser will not render it as markup.
	//nolint:gosec // G705
	fmt.Fprintf(writer, "Request: %s %s\n", requestMethod(request), requestURL(request))
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Details: %s\n", err)
}

// isUpstreamUnreachable reports the network failures that mean "the other side
// is not there", as opposed to a failure of our own.
func isUpstreamUnreachable(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	var opErr *net.OpError

	return errors.As(err, &opErr)
}

func requestMethod(request *http.Request) string {
	if request == nil {
		return ""
	}

	return request.Method
}

func requestURL(request *http.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}

	return request.URL.String()
}
