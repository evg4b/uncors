package mock

import (
	"fmt"
	"net/http"

	"github.com/go-http-utils/headers"
)

func (m *Middleware) serveRawContent(writer http.ResponseWriter) error {
	response := m.response
	header := writer.Header()
	if len(header.Get(headers.ContentType)) == 0 {
		contentType := http.DetectContentType([]byte(response.RawContent))
		header.Set(headers.ContentType, contentType)
	}

	writer.WriteHeader(normaliseCode(response.Code))
	if _, err := fmt.Fprint(writer, response.RawContent); err != nil {
		return fmt.Errorf("failed to write mock content: %w", err)
	}

	return nil
}
