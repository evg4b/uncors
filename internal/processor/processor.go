package processor

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/pterm/pterm"
)

var ErrFailedRequest = errors.New("UNCORS: Failed request handler")

type HandlingMiddleware interface {
	Wrap(next infrastructure.HandlerFunc) infrastructure.HandlerFunc
}

type RequestProcessor struct {
	handlerFunc infrastructure.HandlerFunc
}

func NewRequestProcessor(options ...RequestProcessorOption) *RequestProcessor {
	processor := &RequestProcessor{handlerFunc: finalHandler}

	for i := len(options) - 1; i >= 0; i-- {
		option := options[i]
		option(processor)
	}

	return processor
}

func (rp *RequestProcessor) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	updateRequest(request)

	if err := rp.handlerFunc(response, request); err != nil {
		pterm.Error.Printfln("UNCORS error: %v", err)
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(response, "UNCORS error:", err.Error())
	}
}

func updateRequest(request *http.Request) {
	request.URL.Host = request.Host

	if request.TLS != nil {
		request.URL.Scheme = "https"
	} else {
		request.URL.Scheme = "http"
	}
}

func finalHandler(_ http.ResponseWriter, _ *http.Request) error {
	return ErrFailedRequest
}
