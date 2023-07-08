package config_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestMockClone(t *testing.T) {
	mock := config.Mock{
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

	clonedMock := mock.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &mock, &clonedMock)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.EqualValues(t, mock, clonedMock)
	})

	t.Run("not same headers map", func(t *testing.T) {
		assert.NotSame(t, &mock.Headers, &clonedMock.Headers)
	})

	t.Run("equals headers map", func(t *testing.T) {
		assert.EqualValues(t, mock.Headers, clonedMock.Headers)
	})

	t.Run("not same queries map", func(t *testing.T) {
		assert.NotSame(t, &mock.Queries, &clonedMock.Queries)
	})

	t.Run("equals queries map values", func(t *testing.T) {
		assert.EqualValues(t, mock.Queries, clonedMock.Queries)
	})

	t.Run("not same Response", func(t *testing.T) {
		assert.NotSame(t, &mock.Response, &clonedMock.Response)
	})

	t.Run("equals Response values", func(t *testing.T) {
		assert.EqualValues(t, mock.Response, clonedMock.Response)
	})
}
