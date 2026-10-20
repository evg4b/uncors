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

type Result struct {
	Response *http.Response

	body            []byte
	backendRequests []string
}

func (e *Env) Do(t *testing.T, req *http.Request) *Result {
	t.Helper()

	before := e.Backend.Count()

	resp, err := e.Client.Do(req)
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

func NewRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, url, http.NoBody)
	require.NoError(t, err)

	return req
}

func (r *Result) BodyString() string {
	return string(r.body)
}

func (r *Result) HasBackendRequest() bool {
	return len(r.backendRequests) > 0
}

func (r *Result) BackendRequest(t *testing.T) string {
	t.Helper()
	require.True(t, r.HasBackendRequest(), "no backend request was recorded for this round-trip")

	return Normalize(r.backendRequests[0])
}

func (r *Result) RawBackendRequest(t *testing.T) string {
	t.Helper()
	require.True(t, r.HasBackendRequest(), "no backend request was recorded for this round-trip")

	return r.backendRequests[0]
}

func (r *Result) ResponseDump(t *testing.T) string {
	t.Helper()

	r.Response.Body = io.NopCloser(bytes.NewReader(r.body))

	raw, err := httputil.DumpResponse(r.Response, true)
	require.NoError(t, err)

	return Normalize(string(raw))
}
