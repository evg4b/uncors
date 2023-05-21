// nolint: lll
package infra

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeHTTPClient(t *testing.T) {
	t.Run("return default client where proxy is not set", func(t *testing.T) {
		client, err := MakeHTTPClient("")

		assert.NoError(t, err)
		assert.Equal(t, client, &defaultHTTPClient)
	})

	t.Run("check redirect should return error", func(t *testing.T) {
		t.Run("for default client", func(t *testing.T) {
			client, err := MakeHTTPClient("")

			assert.NoError(t, err)
			err = client.CheckRedirect(nil, nil)
			assert.ErrorIs(t, http.ErrUseLastResponse, err)
		})

		t.Run("for client with proxy", func(t *testing.T) {
			client, err := MakeHTTPClient("http://localhost:8000")

			assert.NoError(t, err)
			err = client.CheckRedirect(nil, nil)
			assert.ErrorIs(t, http.ErrUseLastResponse, err)
		})
	})

	t.Run("return configured client where proxy is set", func(t *testing.T) {
		client, err := MakeHTTPClient("http://localhost:8000")

		assert.NoError(t, err)
		assert.NotEqual(t, client, &defaultHTTPClient)
		assert.NotNil(t, client, &defaultHTTPClient)
	})

	t.Run("return error where urls is incorrect", func(t *testing.T) {
		_, err := MakeHTTPClient("http://loca^host:8000")

		assert.EqualError(t, err, "failed to create http client: parse \"http://loca^host:8000\": invalid character \"^\" in host name")
	})

	t.Run("return error where urls is incorrect", func(t *testing.T) {
		_, err := MakeHTTPClient("http://loca^host:8000")

		assert.EqualError(t, err, "failed to create http client: parse \"http://loca^host:8000\": invalid character \"^\" in host name")
	})
}
