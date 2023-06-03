package helpers_test

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseRequest(t *testing.T) {
	url, err := urlx.Parse(testconstants.HTTPLocalhost)
	testutils.CheckNoError(t, err)

	t.Run("set correct scheme", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				Host: testconstants.Localhost,
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, request.URL.Scheme, "http")
		})

		t.Run("https", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				TLS:  &tls.ConnectionState{},
				Host: testconstants.Localhost,
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, request.URL.Scheme, "https")
		})
	})

	t.Run("fill url.host", func(t *testing.T) {
		request := &http.Request{
			URL:  url,
			TLS:  &tls.ConnectionState{},
			Host: testconstants.Localhost,
		}

		helpers.NormaliseRequest(request)

		assert.Equal(t, request.URL.Host, testconstants.Localhost)
	})
}
