package mock

import (
	"fmt"
	"net/http"
)

type Handler struct {
	mock Mock
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(h.mock.Response.Code)
	fmt.Fprint(writer, h.mock.Response.RawContent)
}
