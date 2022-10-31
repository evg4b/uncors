package testutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func ReadBody(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	response := recorder.Result()
	defer CheckNoError(t, response.Body.Close())

	body, err := io.ReadAll(response.Body)
	CheckNoError(t, err)

	return string(body)
}

func ReadHeader(t *testing.T, recorder *httptest.ResponseRecorder) http.Header {
	t.Helper()

	response := recorder.Result()
	defer CheckNoError(t, response.Body.Close())

	return response.Header
}
