package helpers_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNormaliseRequest(t *testing.T) {
	url, err := urlt.Parse(hosts.Localhost.HTTP().String())
	testutils.CheckNoError(t, err)

	t.Run("set correct scheme", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				Host: hosts.Localhost.Host().String(),
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, "http", request.URL.Scheme)
		})

		t.Run("https", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				TLS:  &tls.ConnectionState{},
				Host: hosts.Localhost.Host().String(),
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, "https", request.URL.Scheme)
		})
	})

	t.Run("fill url.host", func(t *testing.T) {
		request := &http.Request{
			URL:  url,
			TLS:  &tls.ConnectionState{},
			Host: hosts.Localhost.Host().String(),
		}

		helpers.NormaliseRequest(request)

		assert.Equal(t, request.URL.Host, hosts.Localhost.Host().String())
	})
}

func TestIs2xxCode(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusIMUsed} {
		assert.True(t, helpers.Is2xxCode(code), code)
	}

	notOK := []int{http.StatusContinue, http.StatusMultipleChoices, http.StatusBadRequest, http.StatusInternalServerError}
	for _, code := range notOK {
		assert.False(t, helpers.Is2xxCode(code), code)
	}
}

func TestNormaliseStatusCode(t *testing.T) {
	t.Run("return 200 for 0", func(t *testing.T) {
		actual := helpers.NormaliseStatusCode(0)

		assert.Equal(t, http.StatusOK, actual)
	})

	t.Run("return code for non-zero", func(t *testing.T) {
		codes := []int{
			http.StatusContinue,
			http.StatusSwitchingProtocols,
			http.StatusOK,
			http.StatusCreated,
			http.StatusAccepted,
			http.StatusMultipleChoices,
			http.StatusBadRequest,
			http.StatusInternalServerError,
		}

		for _, code := range codes {
			t.Run(fmt.Sprintf("return %d for %d", code, code), func(t *testing.T) {
				actual := helpers.NormaliseStatusCode(code)

				assert.Equal(t, code, actual)
			})
		}
	})
}
