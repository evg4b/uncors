package mock

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/responseprinter"
	"github.com/pterm/pterm"
)

type Handler struct {
	mock       Mock
	mockWriter pterm.PrefixPrinter
}

func NewMockHandler(options ...HandlerOption) *Handler {
	handler := &Handler{
		mockWriter: pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.InfoMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.Style{pterm.FgBlack, pterm.BgLightMagenta},
				Text:  " MOCK  ",
			},
		},
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	updateRequest(request)
	writer.WriteHeader(handler.mock.Response.Code)
	fmt.Fprint(writer, handler.mock.Response.RawContent)
	handler.mockWriter.Println(responseprinter.PrintResponse(&http.Response{
		Request:    request,
		StatusCode: handler.mock.Response.Code,
	}))
}

func updateRequest(request *http.Request) {
	request.URL.Host = request.Host

	if request.TLS != nil {
		request.URL.Scheme = "https"
	} else {
		request.URL.Scheme = "http"
	}
}
