package infra_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const expectedPage = `
███████  ██████   ██████      ███████ ██████  ██████   ██████  ██████  
██      ██  ████ ██  ████     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████ ██ ██ ██ ██ ██ ██     █████   ██████  ██████  ██    ██ ██████  
     ██ ████  ██ ████  ██     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████  ██████   ██████      ███████ ██   ██ ██   ██  ██████  ██   ██ 


Occurred error: net/http: abort Handler
`

func TestHttpError(t *testing.T) {
	recorder := httptest.NewRecorder()
	infra.HTTPError(recorder, http.ErrAbortHandler)
	body := testutils.ReadBody(t, recorder)

	t.Run("write correct page", func(t *testing.T) {
		assert.Contains(t, body, expectedPage)
	})

	t.Run("write correct headers", func(t *testing.T) {
		header := recorder.Header()

		assert.NotNil(t, header[headers.ContentType])
		assert.NotNil(t, header[headers.ContentEncoding])
		assert.NotNil(t, header[headers.CacheControl])
		assert.NotNil(t, header[headers.Pragma])
		assert.NotNil(t, header[headers.XContentTypeOptions])
		assert.Nil(t, header[headers.SetCookie])
	})

	t.Run("Should include stack trace", func(t *testing.T) {
		assert.Regexp(t, "Stack trace: goroutine \\d+ \\[running\\]:", body)
	})

	t.Run("Should include memory usage", func(t *testing.T) {
		assert.Contains(t, body, "Memory usage:")
		assert.Regexp(t, "Alloc = [\\d\\.]+ [Mk]B", body)
		assert.Regexp(t, "TotalAlloc = [\\d\\.]+ [Mk]B", body)
		assert.Regexp(t, "Sys = [\\d\\.]+ [Mk]B", body)
		assert.Regexp(t, "NumGC = [\\d\\.]+", body)
	})
}
