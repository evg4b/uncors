//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/stretchr/testify/require"
)

// Result captures both sides of a single proxy round-trip:
// the raw request the backend received and the response the client got back.
type Result struct {
	// Response is the HTTP response received by the client.
	// Its body has been drained and buffered; close or re-read it freely.
	Response *http.Response

	body            []byte
	backendRequests []string
}

// Do sends req through the proxy client and captures the backend request(s)
// that arrived since the previous Do call (or since the last Backend.Reset).
// It reads and buffers the response body so Result.BodyString and
// Result.ResponseDump are available without an extra Read.
func (e *Env) Do(t *testing.T, req *http.Request) *Result {
	t.Helper()

	before := e.Backend.Count()

	resp, err := e.Client.Do(req) //nolint:gosec // G704: requests target the in-process test proxy
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	all := e.Backend.Requests()
	newReqs := all[before:]

	snapshot := make([]string, len(newReqs))
	copy(snapshot, newReqs)

	return &Result{
		Response:        resp,
		body:            body,
		backendRequests: snapshot,
	}
}

// NewRequest builds an HTTP request with a background context.
func NewRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	require.NoError(t, err)

	return req
}

// BodyString returns the response body as a string.
func (r *Result) BodyString() string {
	return string(r.body)
}

// HasBackendRequest reports whether the backend received at least one new
// request during this round-trip (i.e. the proxy forwarded rather than mocked).
func (r *Result) HasBackendRequest() bool {
	return len(r.backendRequests) > 0
}

// BackendRequest returns the normalized raw dump of the first backend request
// received during this round-trip, suitable for snapshot assertions.
// It fails the test when no backend request was recorded.
func (r *Result) BackendRequest(t *testing.T) string {
	t.Helper()
	require.True(t, r.HasBackendRequest(), "no backend request was recorded for this round-trip")

	return Normalize(r.backendRequests[0])
}

// ResponseDump returns a normalized full HTTP response dump (status line,
// headers, body) suitable for snapshot assertions.
func (r *Result) ResponseDump(t *testing.T) string {
	t.Helper()

	r.Response.Body = io.NopCloser(bytes.NewReader(r.body))

	raw, err := httputil.DumpResponse(r.Response, true)
	require.NoError(t, err)

	return Normalize(string(raw))
}
