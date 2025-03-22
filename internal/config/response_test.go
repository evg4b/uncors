package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestResponseClone(t *testing.T) {
	response := config.Response{
		Code: http.StatusOK,
		Headers: map[string]string{
			headers.ContentType:  "plain/text",
			headers.CacheControl: "none",
		},
		Raw:   "this is plain text",
		File:  "~/projects/uncors/response/demo.json",
		Delay: time.Hour,
		Seed:  8123,
	}

	clonedResponse := response.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &response, &clonedResponse)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, response, clonedResponse)
	})

	t.Run("not same Headers map", func(t *testing.T) {
		assert.NotSame(t, &response.Headers, &clonedResponse.Headers)
	})

	t.Run("response type", func(t *testing.T) {
		t.Run("raw response", func(t *testing.T) {
			response := config.Response{
				Code: http.StatusOK,
				Raw:  "this is plain text",
			}

			t.Run("IsRaw", func(t *testing.T) {
				assert.True(t, response.IsRaw())
			})

			t.Run("IsFile", func(t *testing.T) {
				assert.False(t, response.IsFile())
			})

			t.Run("IsFake", func(t *testing.T) {
				assert.False(t, response.IsFake())
			})
		})

		t.Run("file response", func(t *testing.T) {
			response := config.Response{
				Code: http.StatusOK,
				File: "~/projects/uncors/response/demo.json",
			}

			t.Run("IsRaw", func(t *testing.T) {
				assert.False(t, response.IsRaw())
			})

			t.Run("IsFile", func(t *testing.T) {
				assert.True(t, response.IsFile())
			})

			t.Run("IsFake", func(t *testing.T) {
				assert.False(t, response.IsFake())
			})
		})

		t.Run("file response", func(t *testing.T) {
			response := config.Response{
				Code: http.StatusOK,
				Fake: &fakedata.Node{Type: "sentence"},
			}

			t.Run("IsRaw", func(t *testing.T) {
				assert.False(t, response.IsRaw())
			})

			t.Run("IsFile", func(t *testing.T) {
				assert.False(t, response.IsFile())
			})

			t.Run("IsFake", func(t *testing.T) {
				assert.True(t, response.IsFake())
			})
		})
	})
}
