package infra

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeHTTPClient(t *testing.T) {
	t.Run("return client with default transport where proxy is not set", func(t *testing.T) {
		client := MakeHTTPClient("")

		assert.NotNil(t, client)
		assert.NotNil(t, client.Transport)
		assert.Equal(t, defaultTimeout, client.Timeout)
	})

	t.Run("check redirect should return error", func(t *testing.T) {
		t.Run("for default client", func(t *testing.T) {
			client := MakeHTTPClient("")

			err := client.CheckRedirect(nil, nil)
			require.ErrorIs(t, http.ErrUseLastResponse, err)
		})

		t.Run("for client with proxy", func(t *testing.T) {
			client := MakeHTTPClient("http://localhost:8000")

			err := client.CheckRedirect(nil, nil)
			require.ErrorIs(t, http.ErrUseLastResponse, err)
		})
	})

	t.Run("return configured client where proxy is set", func(t *testing.T) {
		client := MakeHTTPClient("http://localhost:8000")

		assert.NotNil(t, client)
		assert.NotNil(t, client.Transport)
		assert.Equal(t, defaultTimeout, client.Timeout)
	})

	t.Run("return error where url is incorrect", func(t *testing.T) {
		expectedError := "failed to create http client: parse \"http://loca^host:8000\": invalid character \"^\" in host name"
		assert.PanicsWithError(t, expectedError, func() {
			MakeHTTPClient("http://loca^host:8000")
		})
	})
}
