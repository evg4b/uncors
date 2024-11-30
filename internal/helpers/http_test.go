package helpers_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	code     int
	expected bool
}

func describe(c testCase) string {
	return fmt.Sprintf("should return %t for code %d", c.expected, c.code)
}

func TestNormaliseRequest(t *testing.T) {
	url, err := urlx.Parse(hosts.Localhost.HTTP())
	testutils.CheckNoError(t, err)

	t.Run("set correct scheme", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				Host: hosts.Localhost.Host(),
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, "http", request.URL.Scheme)
		})

		t.Run("https", func(t *testing.T) {
			request := &http.Request{
				URL:  url,
				TLS:  &tls.ConnectionState{},
				Host: hosts.Localhost.Host(),
			}

			helpers.NormaliseRequest(request)

			assert.Equal(t, "https", request.URL.Scheme)
		})
	})

	t.Run("fill url.host", func(t *testing.T) {
		request := &http.Request{
			URL:  url,
			TLS:  &tls.ConnectionState{},
			Host: hosts.Localhost.Host(),
		}

		helpers.NormaliseRequest(request)

		assert.Equal(t, request.URL.Host, hosts.Localhost.Host())
	})
}

func TestIs1xxCode(t *testing.T) {
	cases := []testCase{
		{http.StatusContinue, true},
		{http.StatusSwitchingProtocols, true},
		{http.StatusOK, false},
		{http.StatusMovedPermanently, false},
		{http.StatusBadRequest, false},
		{http.StatusInternalServerError, false},
	}

	for _, c := range cases {
		t.Run(describe(c), func(t *testing.T) {
			actual := helpers.Is1xxCode(c.code)
			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestIs2xxCode(t *testing.T) {
	cases := []testCase{
		{http.StatusOK, true},
		{http.StatusCreated, true},
		{http.StatusAccepted, true},
		{http.StatusSwitchingProtocols, false},
		{http.StatusMultipleChoices, false},
		{http.StatusBadRequest, false},
		{http.StatusInternalServerError, false},
	}

	for _, c := range cases {
		t.Run(describe(c), func(t *testing.T) {
			actual := helpers.Is2xxCode(c.code)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestIs3xxCode(t *testing.T) {
	cases := []testCase{
		{http.StatusMultipleChoices, true},
		{http.StatusMovedPermanently, true},
		{http.StatusFound, true},
		{http.StatusOK, false},
		{http.StatusSwitchingProtocols, false},
		{http.StatusBadRequest, false},
		{http.StatusInternalServerError, false},
	}

	for _, c := range cases {
		t.Run(describe(c), func(t *testing.T) {
			actual := helpers.Is3xxCode(c.code)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestIs4xxCode(t *testing.T) {
	cases := []testCase{
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusForbidden, true},
		{http.StatusOK, false},
		{http.StatusSwitchingProtocols, false},
		{http.StatusMultipleChoices, false},
		{http.StatusInternalServerError, false},
	}

	for _, c := range cases {
		t.Run(describe(c), func(t *testing.T) {
			actual := helpers.Is4xxCode(c.code)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestIs5xxCode(t *testing.T) {
	cases := []testCase{
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusOK, false},
		{http.StatusSwitchingProtocols, false},
		{http.StatusMultipleChoices, false},
		{http.StatusInternalServerError, true},
		{http.StatusNetworkAuthenticationRequired, true},
		{http.StatusHTTPVersionNotSupported, true},
	}

	for _, c := range cases {
		t.Run(describe(c), func(t *testing.T) {
			actual := helpers.Is5xxCode(c.code)

			assert.Equal(t, c.expected, actual)
		})
	}
}
