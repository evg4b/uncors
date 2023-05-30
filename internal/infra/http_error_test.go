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

	t.Run("write correct page", func(t *testing.T) {
		assert.Equal(t, expectedPage, testutils.ReadBody(t, recorder))
	})

	t.Run("write correct headers", func(t *testing.T) {
		header := recorder.Header()

		assert.NotNil(t, header[headers.ContentType])
		assert.NotNil(t, header[headers.XContentTypeOptions])
	})
}
