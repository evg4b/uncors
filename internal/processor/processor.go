package processor

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/pterm/pterm"
)

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

func (rp *RequestProcessor) HandleRequest(w http.ResponseWriter, req *http.Request) {
	err := rp.handlerFunc(w, req)
	if err != nil {
		pterm.Error.Printfln("UNCORS error: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "UNCORS error:")
		fmt.Fprintln(w, err.Error())
	}
}

func finalFandler(w http.ResponseWriter, r *http.Request) error {
	return errors.New("UNCORS: Failed requset handler")
}
