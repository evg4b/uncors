package testutils

import (
	"io"
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
