package testutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
)

func ReadBody(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	response := recorder.Result()
	defer helpers.CloseSafe(response.Body)

	body, err := io.ReadAll(response.Body)
	CheckNoError(t, err)

	return string(body)
}

func ReadHeader(t *testing.T, recorder *httptest.ResponseRecorder) http.Header {
	t.Helper()

	response := recorder.Result()
	defer helpers.CloseSafe(response.Body)

	return response.Header
}
