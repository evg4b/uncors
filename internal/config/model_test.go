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
	object := config.Response{
		Code: http.StatusOK,
		Headers: map[string]string{
			headers.ContentType:  "plain/text",
			headers.CacheControl: "none",
		},
		Raw:   "this is plain text",
		File:  "~/projects/uncors/response/demo.json",
		Delay: time.Hour,
	}

	actual := object.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &object, &actual)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, object, actual)
	})

	t.Run("not same Headers map", func(t *testing.T) {
		assert.NotSame(t, &object.Headers, &actual.Headers)
	})
}

func TestMockClone(t *testing.T) {
	object := config.Mock{
		Path:   "/constants",
		Method: http.MethodGet,
		Queries: map[string]string{
			"page": "10",
			"size": "50",
		},
		Headers: map[string]string{
			headers.ContentType:  "plain/text",
			headers.CacheControl: "none",
		},
		Response: config.Response{
			Code: http.StatusOK,
			Raw:  `{ "status": "ok" }`,
		},
	}

	actual := object.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &object, &actual)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, object, actual)
	})

	t.Run("not same Headers map", func(t *testing.T) {
		assert.NotSame(t, &object.Headers, &actual.Headers)
	})

	t.Run("not same Queries map", func(t *testing.T) {
		assert.NotSame(t, &object.Headers, &actual.Headers)
	})

	t.Run("not same Response", func(t *testing.T) {
		assert.NotSame(t, &object.Response, &actual.Response)
	})
}
