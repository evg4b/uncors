package processor

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/pterm/pterm"
)

var ErrFailedRequset = errors.New("UNCORS: Failed requset handler")

type HandlingMiddleware interface {
	Wrap(next infrastructure.HandlerFunc) infrastructure.HandlerFunc
}

type RequestProcessor struct {
	handlerFunc infrastructure.HandlerFunc
}

func NewRequestProcessor(options ...requestProcessorOption) *RequestProcessor {
	processor := &RequestProcessor{handlerFunc: finalFandler}

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

func finalFandler(w http.ResponseWriter, r *http.Request) error {
	return ErrFailedRequset
}
