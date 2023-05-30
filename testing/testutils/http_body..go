package testutils

import (
	"github.com/evg4b/uncors/internal/helpers"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
