package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"

	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestUncorsRequestHandler(t *testing.T) {
	fs := testutils.FsFromMap(t, map[string]string{
		"/images/background.png": "background.png",
		"/images/svg/icons.svg":  "icons.svg",
		"/assets/js/index.js":    "index.js",
		"/assets/css/styles.css": "styles.css",
		"/assets/index.html":     "index.html",
		"/mock.json":             "mock.json",
	})

	mappings := []configuration.URLMapping{
		{
			From: "http://localhost",
			To:   "https://localhost",
			Statics: []configuration.StaticDirMapping{
				{Dir: "/assets", Path: "/cc/", Index: "index.html"},
				{Dir: "/assets", Path: "/pnp/", Index: "index.php"},
				{Dir: "/images", Path: "/img/"},
			},
		},
	}

	mockDefs := []configuration.Mock{
		{
			Path: "/mocks/1",
			Response: configuration.Response{
				Code:       http.StatusOK,
				RawContent: "mocks-1",
			},
		},
		{
			Path: "/mocks/2",
			Response: configuration.Response{
				Code: http.StatusOK,
				File: "/mock.json",
			},
		},
	}

	factory, err := urlreplacer.NewURLReplacerFactory(mappings)
	testutils.CheckNoError(t, err)

	hand := handler.NewUncorsRequestHandler(
		handler.WithLogger(mocks.NewLoggerMock(t)),
		handler.WithMocks(mockDefs),
		handler.WithFileSystem(fs),
		handler.WithURLReplacerFactory(factory),
		handler.WithHTTPClient(mocks.NewHTTPClientMock(t)),
		handler.WithMappings(mappings),
	)

	t.Run("statics directory", func(t *testing.T) {
		t.Run("with index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     "index.html",
						url:      "http://localhost/cc/index.html",
						expected: "index.html",
					},
					{
						name:     "index.js",
						url:      "http://localhost/cc/js/index.js",
						expected: "index.js",
					},
					{
						name:     "styles.css",
						url:      "http://localhost/cc/css/styles.css",
						expected: "styles.css",
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, 200, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})

			t.Run("should return index file by default", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/cc/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "index.html", testutils.ReadBody(t, recorder))
			})

			t.Run("should return error code when index file doesn't exists", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "http://localhost/pnp/unknown.html", nil)
				helpers.NormaliseRequest(request)

				hand.ServeHTTP(recorder, request)

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, testutils.ReadBody(t, recorder), "Internal Server Error")
			})
		})

		t.Run("without index file", func(t *testing.T) {
			t.Run("should return static file", func(t *testing.T) {
				tests := []struct {
					name     string
					url      string
					expected string
				}{
					{
						name:     "background.png",
						url:      "http://localhost/img/background.png",
						expected: "background.png",
					},
					{
						name:     "icons.svg",
						url:      "http://localhost/img/svg/icons.svg",
						expected: "icons.svg",
					},
				}
				for _, testCase := range tests {
					t.Run(testCase.name, func(t *testing.T) {
						recorder := httptest.NewRecorder()
						request := httptest.NewRequest(http.MethodGet, testCase.url, nil)
						helpers.NormaliseRequest(request)

						hand.ServeHTTP(recorder, request)

						assert.Equal(t, http.StatusOK, recorder.Code)
						assert.Equal(t, testCase.expected, testutils.ReadBody(t, recorder))
					})
				}
			})
		})
	})
}
