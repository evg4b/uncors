package contracts

import (
	"net/http"
	"time"
)

type ResponseCapture struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

type BodyCapturer interface {
	EnableBodyCapture()
	Captured() ResponseCapture
}

type ResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
	BodyCapturer
}
