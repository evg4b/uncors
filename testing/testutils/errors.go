package testutils

import "testing"

func CheckNoError(t *testing.T, err error) {
	if t != nil {
		t.Helper()
	}

	if err != nil {
		t.Fatal(err)
	}
}

