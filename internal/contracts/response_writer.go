package contracts

import "net/http"

type ResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
	BodyCapturer
}
