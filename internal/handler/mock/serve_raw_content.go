package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/go-http-utils/headers"
)

func (m *Middleware) serveRawContent(writer http.ResponseWriter) {
	response := m.response
	header := writer.Header()
	if len(header.Get(headers.ContentType)) == 0 {
		contentType := http.DetectContentType([]byte(response.Raw))
		header.Set(headers.ContentType, contentType)
	}

	writer.WriteHeader(normaliseCode(response.Code))
	sfmt.Fprint(writer, response.Raw)
}
