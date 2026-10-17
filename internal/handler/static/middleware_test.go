package static_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	indexJS   = "/assets/index.js"
	demoJS    = "/assets/demo.js"
	indexHTML = "/index.html"
)

const (
	indexJSContent   = "console.log('index.js')"
	demoJSContent    = "console.log('demo.js')"
	indexHTMLContent = "<html!></html>"
)

func newStaticFs(t *testing.T) afero.Fs {
	t.Helper()

	return testutils.FsFromMap(t, map[string]string{
		indexJS:   indexJSContent,
		demoJS:    demoJSContent,
		indexHTML: indexHTMLContent,
	})
}

func TestStaticMiddleware(t *testing.T) {
	fs := newStaticFs(t)

	staticFileTests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "should send pain file",
			path:     indexHTML,
			expected: indexHTMLContent,
		},
		{
			name:     "should send file from folder",
			path:     indexJS,
			expected: indexJSContent,
		},
		{
			name:     "should send file ignore query params",
			path:     indexHTML + "?testParam=test",
			expected: indexHTMLContent,
		},
		{
			name:     "should send file from folder ignore query params",
			path:     demoJS + "?testParam=test",
			expected: demoJSContent,
		},
		{
			name:     "should send pain file without root slash",
			path:     strings.TrimPrefix(indexHTML, "/"),
			expected: indexHTMLContent,
		},
		{
			name:     "should send demo.js file from folder without root slash",
			path:     strings.TrimPrefix(demoJS, "/"),
			expected: demoJSContent,
		},
	}

	notExistingFilesTests := []struct {
		name string
		path string
	}{
		{
			name: "where file not exists",
			path: "/options.html",
		},
		{
			name: "where request directory",
			path: "/assets/",
		},
		{
			name: "where request directory without trailing slash",
			path: "/assets",
		},
		{
			name: "where request not exists directory",
			path: "/options/",
		},
	}

	t.Run("index file is not configured", func(t *testing.T) {
		const (
			testHTTPStatusCode = 999
			testHTTPBody       = "this is tests response"
		)

		handler := static.NewStaticMiddleware(static.WithFileSystem(fs)).
			Wrap(infra.HandlerFunc(func(writer contracts.ResponseWriter, _ *contracts.Request) error {
				writer.WriteHeader(testHTTPStatusCode)
				fmt.Fprint(writer, testHTTPBody)

				return nil
			}))

		for _, testCase := range staticFileTests {
			t.Run(testCase.name, func(t *testing.T) {
				expectServed(t, handler, testCase.path, http.StatusOK, testCase.expected)
			})
		}

		for _, testCase := range notExistingFilesTests {
			t.Run("forward "+testCase.name, func(t *testing.T) {
				expectServed(t, handler, testCase.path, testHTTPStatusCode, testHTTPBody)
			})
		}
	})

	t.Run("index file is configured", func(t *testing.T) {
		handler := static.NewStaticMiddleware(static.WithFileSystem(fs), static.WithIndex(indexHTML)).
			Wrap(mocks.FailNowHandlerMock(t))

		for _, testCase := range staticFileTests {
			t.Run(testCase.name, func(t *testing.T) {
				expectServed(t, handler, testCase.path, http.StatusOK, testCase.expected)
			})
		}

		for _, testCase := range notExistingFilesTests {
			t.Run("index "+testCase.name, func(t *testing.T) {
				expectServed(t, handler, testCase.path, http.StatusOK, indexHTMLContent)
			})
		}
	})

	t.Run("index file doesn't exists", func(t *testing.T) {
		handler := static.NewStaticMiddleware(static.WithFileSystem(fs), static.WithIndex("/not-exists.html")).
			Wrap(mocks.FailNowHandlerMock(t))

		_, err := serveStatic(t, handler, "/options/")

		assert.Error(t, err)
	})
}

func serveStatic(t *testing.T, handler contracts.Handler, path string) (*httptest.ResponseRecorder, error) {
	t.Helper()

	requestURI, err := url.Parse(path)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()

	return recorder, handler.ServeHTTP(server.NewResponseRecorder(recorder), &http.Request{
		Method: http.MethodGet,
		URL:    requestURI,
	})
}

func expectServed(t *testing.T, handler contracts.Handler, path string, code int, body string) {
	t.Helper()

	recorder, err := serveStatic(t, handler, path)
	require.NoError(t, err)

	assert.Equal(t, code, recorder.Code)
	assert.Equal(t, body, testutils.ReadBody(t, recorder))
}

func serveOnce(middleware *static.Middleware, requestURL *url.URL) {
	handler := middleware.Wrap(infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *contracts.Request) error {
		return nil
	}))

	_ = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), &http.Request{
		Method: http.MethodGet,
		URL:    requestURL,
	})
}

func TestStaticMiddlewareReleasesFileHandles(t *testing.T) {
	fs := newStaticFs(t)

	tests := []struct {
		name  string
		index string
		path  string
	}{
		{name: "file", index: indexHTML, path: indexJS},
		{name: "directory served by index", index: indexHTML, path: "/assets/"},
		{name: "missing file served by index", index: indexHTML, path: "/missing.js"},
		{name: "directory without index", index: "", path: "/assets/"},
		{name: "directory with missing index", index: "/not-exists.html", path: "/assets/"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			trackingFs := testutils.NewTrackingFs(fs)
			middleware := static.NewStaticMiddleware(
				static.WithFileSystem(trackingFs),
				static.WithIndex(testCase.index),
			)

			requestURI, err := url.Parse(testCase.path)
			require.NoError(t, err)

			serveOnce(middleware, requestURI)

			assert.Zero(t, trackingFs.OpenFiles(), "file handles were left open")
		})
	}
}

func TestStaticMiddlewareStaysInsideRoot(t *testing.T) {
	fs := newStaticFs(t)

	for _, rawPath := range []string{
		"/../../etc/passwd",
		"/assets/../../../etc/passwd",
		"../etc/passwd",
		"/static/../../etc/passwd",
	} {
		t.Run(rawPath, func(t *testing.T) {
			trackingFs := testutils.NewTrackingFs(fs)
			middleware := static.NewStaticMiddleware(
				static.WithFileSystem(trackingFs),
				static.WithPrefix("/static"),
			)

			serveOnce(middleware, &url.URL{Path: rawPath})

			opened := trackingFs.Opened()
			require.NotEmpty(t, opened)

			for _, name := range opened {
				assert.NotContains(t, name, "..")
				assert.True(t, strings.HasPrefix(filepath.ToSlash(name), "/"), name)
			}
		})
	}
}

func TestStaticMiddlewarePathUnderFile(t *testing.T) {
	const (
		forwardCode = 999
		forwardBody = "forwarded"
	)

	osFs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	for name, content := range map[string]string{
		indexJS:   indexJSContent,
		demoJS:    demoJSContent,
		indexHTML: indexHTMLContent,
	} {
		require.NoError(t, osFs.MkdirAll(filepath.Dir(name), 0o755))
		require.NoError(t, afero.WriteFile(osFs, name, []byte(content), 0o644))
	}

	fileSystems := map[string]afero.Fs{
		"MemMapFs":       newStaticFs(t),
		"OsFs in a base": osFs,
	}

	paths := []string{
		indexHTML + "/foo",
		indexJS + "/foo/bar",
	}

	for fsName, fileSystem := range fileSystems {
		t.Run(fsName, func(t *testing.T) {
			forward := static.NewStaticMiddleware(static.WithFileSystem(fileSystem)).
				Wrap(infra.HandlerFunc(func(writer contracts.ResponseWriter, _ *contracts.Request) error {
					writer.WriteHeader(forwardCode)
					fmt.Fprint(writer, forwardBody)

					return nil
				}))

			index := static.NewStaticMiddleware(static.WithFileSystem(fileSystem), static.WithIndex(indexHTML)).
				Wrap(mocks.FailNowHandlerMock(t))

			for _, path := range paths {
				t.Run("forward "+path, func(t *testing.T) {
					expectServed(t, forward, path, forwardCode, forwardBody)
				})

				t.Run("index "+path, func(t *testing.T) {
					expectServed(t, index, path, http.StatusOK, indexHTMLContent)
				})
			}
		})
	}
}
