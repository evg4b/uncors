package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
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
}
