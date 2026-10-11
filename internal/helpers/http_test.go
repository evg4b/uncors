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

func TestStatusCodeClasses(t *testing.T) {
	checks := map[string]func(int) bool{
		"1xx": helpers.Is1xxCode,
		"2xx": helpers.Is2xxCode,
		"3xx": helpers.Is3xxCode,
		"4xx": helpers.Is4xxCode,
		"5xx": helpers.Is5xxCode,
	}

	codes := map[string][]int{
		"1xx": {http.StatusContinue, http.StatusSwitchingProtocols},
		"2xx": {http.StatusOK, http.StatusCreated, http.StatusAccepted},
		"3xx": {http.StatusMultipleChoices, http.StatusMovedPermanently, http.StatusFound},
		"4xx": {http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
		"5xx": {http.StatusInternalServerError, http.StatusHTTPVersionNotSupported, http.StatusNetworkAuthenticationRequired},
	}

	for class, check := range checks {
		for codeClass, classCodes := range codes {
			for _, code := range classCodes {
				assert.Equal(t, class == codeClass, check(code), "%s check for %d", class, code)
			}
		}
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
