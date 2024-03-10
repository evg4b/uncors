package request_tracker

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type HttpRequestTracker struct {
	tracker RequestTracker
	client  contracts.HTTPClient
	prefix  string
}

func (h HttpRequestTracker) Do(req *http.Request) (*http.Response, error) {
	id := h.tracker.RegisterRequest(req, h.prefix)
	resp, err := h.client.Do(req)
	if err != nil {
		h.tracker.ResolveRequest(id, 0)
		return nil, err
	}

	h.tracker.ResolveRequest(id, resp.StatusCode)

	return resp, nil
}
