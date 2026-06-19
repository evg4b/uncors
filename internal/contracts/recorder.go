package contracts

import (
	"net/http"
	"time"
)

// ResponseCapture holds the captured response data after a request completes.
type ResponseCapture struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

// BodyCapturer is implemented by ResponseRecorder; middleware use it to opt in
// to body buffering and read the captured response after calling next.
type BodyCapturer interface {
	EnableBodyCapture()
	Captured() ResponseCapture
}
