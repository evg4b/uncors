package contracts

import (
	"net/http"
	"time"
)

type ResponseCapture struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	// Truncated reports that the response was larger than the capture limit, so
	// Body holds only its beginning.
	Truncated bool
	Duration  time.Duration
}

type BodyCapturer interface {
	// EnableBodyCapture starts recording the response body, keeping at most
	// maxBytes of it. Capture is skipped for responses that are meant to stream.
	EnableBodyCapture(maxBytes int64)
	Captured() ResponseCapture
}
