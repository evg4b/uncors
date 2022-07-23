package infrastructure_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeHTTPReqDecorator(t *testing.T) {
	expectedHost := "localhost:3000"
	expectedScheme := "http"
	hadnlerFunc := infrastructure.NormalizeHTTPReqDecorator(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedHost, r.URL.Host)
		assert.Equal(t, expectedScheme, r.URL.Scheme)
	})

	req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
	req.Host = expectedHost
	if err != nil {
		t.Fatal(err)
	}

	http.HandlerFunc(hadnlerFunc).
		ServeHTTP(httptest.NewRecorder(), req)
}

func TestNormalizeNormalizeHTTPSReqDecorator(t *testing.T) {
	expectedHost := "localhost:3000"
	expectedScheme := "https"
	hadnlerFunc := infrastructure.NormalizeHTTPSReqDecorator(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedHost, r.URL.Host)
		assert.Equal(t, expectedScheme, r.URL.Scheme)
	})

	req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
	req.Host = expectedHost
	if err != nil {
		t.Fatal(err)
	}

	http.HandlerFunc(hadnlerFunc).
		ServeHTTP(httptest.NewRecorder(), req)
}
