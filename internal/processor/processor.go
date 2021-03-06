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

func (rp *RequestProcessor) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if err := rp.handlerFunc(w, r); err != nil {
		pterm.Error.Printfln("UNCORS error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "UNCORS error:", err.Error())
	}
}

func finalFandler(w http.ResponseWriter, r *http.Request) error {
	return ErrFailedRequset
}
