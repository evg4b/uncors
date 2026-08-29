package testutils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(responseCreator RoundTripFunc) *http.Client {
	return &http.Client{Transport: responseCreator}
}

func ReadBody(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	response := recorder.Result()
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	CheckNoError(t, err)

	return string(body)
}

func ReadHeader(t *testing.T, recorder *httptest.ResponseRecorder) http.Header {
	t.Helper()

	response := recorder.Result()
	defer response.Body.Close()

	return response.Header
}
