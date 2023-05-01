package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-http-utils/headers"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/middlewares/mock"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareStaticRoutes(t *testing.T) {
	logger := mocks.NewNoopLogger(t)

	middleware := mock.NewMockMiddleware(
		mock.WithLogger(logger),
		mock.WithFileSystem(testutils.FsFromMap(t, map[string]string{
			"cc/demo.json":       `{ "prop1": 1 }`,
			"cc/demo.txt":        `txt file`,
			"other/default.html": `<html><body>test</body></html>`,
		})),
		mock.WithMappings([]configuration.URLMapping{
			{Statics: []configuration.StaticDirMapping{
				{Path: "/cc", Dir: "cc"},
				{Path: "/static", Dir: "cc"},
				{Path: "/lorem", Dir: "other"},
			}},
		}),
	)

	t.Run("file content serving", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "server txt file",
				path:     "/cc/demo.txt",
				expected: `txt file`,
			},
			{
				name:     "server json file",
				path:     "/cc/demo.json",
				expected: `{ "prop1": 1 }`,
			},
			{
				name:     "server txt file for other mapping",
				path:     "/static/demo.txt",
				expected: `txt file`,
			},
			{
				name:     "server json file for other mapping",
				path:     "/static/demo.json",
				expected: `{ "prop1": 1 }`,
			},
			{
				name:     "server html file",
				path:     "/lorem/default.html",
				expected: `<html><body>test</body></html>`,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, testCase.path, nil)
				middleware.ServeHTTP(rec, r)
				body := testutils.ReadBody(t, rec)

				assert.Equal(t, testCase.expected, body)
				assert.Equal(t, http.StatusOK, rec.Code)
			})
		}
	})

	t.Run("redirects", func(t *testing.T) {
		t.Run("dir redirect", func(t *testing.T) {
			tests := []struct {
				name     string
				path     string
				expected string
			}{
				{
					name:     "cc prefix",
					path:     "/cc",
					expected: "/cc/",
				},
				{
					name:     "static prefix",
					path:     "/static",
					expected: "/static/",
				},
				{
					name:     "lorem prefix",
					path:     "/lorem",
					expected: "/lorem/",
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					rec := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, testCase.path, nil)
					middleware.ServeHTTP(rec, r)

					assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
					assert.Equal(t, testCase.expected, rec.Header().Get(headers.Location))
				})
			}
		})

		t.Run("no redirects for dir with trailing slash redirect", func(t *testing.T) {
			tests := []struct {
				name string
				path string
			}{
				{
					name: "cc prefix",
					path: "/cc/",
				},
				{
					name: "static prefix",
					path: "/static/",
				},
				{
					name: "lorem prefix",
					path: "/lorem/",
				},
			}
			for _, testCase := range tests {
				t.Run(testCase.name, func(t *testing.T) {
					rec := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, testCase.path, nil)
					middleware.ServeHTTP(rec, r)

					assert.Equal(t, http.StatusOK, rec.Code)
					assert.NotContains(t, rec.Header(), headers.Location)
				})
			}
		})
	})
}
