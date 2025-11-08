package mock

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/go-http-utils/headers"
)

func (h *Handler) serveRawContent(writer http.ResponseWriter) error {
	response := h.response

	header := writer.Header()
	if len(header.Get(headers.ContentType)) == 0 {
		contentType := http.DetectContentType([]byte(response.Raw))
		header.Set(headers.ContentType, contentType)
	}

	writer.WriteHeader(helpers.NormaliseStatusCode(response.Code))
	_, err := fmt.Fprint(writer, response.Raw)

	return err
}
