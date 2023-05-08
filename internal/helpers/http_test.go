package helpers_test

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseRequest(t *testing.T) {
	var url, err = urlx.Parse("http://localhost")
	testutils.CheckNoError(t, err)

	t.Run("set correct scheme", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			var request = &http.Request{
				URL:  url,
				Host: "localhost",
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, request.URL.Scheme, "http")
		})

		t.Run("https", func(t *testing.T) {
			var request = &http.Request{
				URL:  url,
				TLS:  &tls.ConnectionState{},
				Host: "localhost",
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, request.URL.Scheme, "https")
		})
	})

	t.Run("fill url.host", func(t *testing.T) {
		var request = &http.Request{
			URL:  url,
			TLS:  &tls.ConnectionState{},
			Host: "localhost",
		}

		helpers.NormaliseRequest(request)

		assert.Equal(t, request.URL.Host, "localhost")
	})
}
