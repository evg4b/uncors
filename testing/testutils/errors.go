package testutils

import (
	"errors"
	"net/http"
	"testing"
)

func CheckNoError(t *testing.T, err error) {
	if t != nil {
		t.Helper()
	}

	if err != nil {
		t.Fatal(err)
	}
}

func CheckNoServerError(t *testing.T, err error) {
	t.Helper()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
}
