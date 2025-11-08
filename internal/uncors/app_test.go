package uncors_test

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/evg4b/uncors/testing/testutils/appbuilder"
	"github.com/phayes/freeport"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const delay = 10 * time.Millisecond

func TestUncorsApp(t *testing.T) {
	ctx := t.Context()
	fs := afero.NewMemMapFs()
	expectedResponse := "UNCORS OK!"

	t.Run("handle request", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		t.Skip()
		t.Run("HTTP", func(t *testing.T) {
			port := freeport.GetPort()
			appBuilder := appbuilder.NewAppBuilder(t).
				WithFs(fs)

			uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(expectedResponse),
					},
				},
			})

			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, http.DefaultClient, appBuilder.URI())

			assert.Equal(t, expectedResponse, response)
		})

		t.Run("HTTPS", func(t *testing.T) {
			httpClient := testutils.SetupHTTPSTest(t, fs) // nolint: staticcheck
			port := freeport.GetPort()
			appBuilder := appbuilder.NewAppBuilder(t).
				WithFs(fs).
				WithHTTPS()

			uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPSPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(expectedResponse),
					},
				},
			})

			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, httpClient, appBuilder.URI())

			assert.Equal(t, expectedResponse, response)
		})
	}))

	t.Run("restart server", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		t.Skip()

		const otherExpectedRepose = `{ "bla": true }`

		t.Run("HTTP", func(t *testing.T) {
			port := freeport.GetPort()
			appBuilder := appbuilder.NewAppBuilder(t).
				WithFs(fs)

			uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(expectedResponse),
					},
				},
			})

			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, http.DefaultClient, appBuilder.URI())
			assert.Equal(t, expectedResponse, response)

			uncorsApp.Restart(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(otherExpectedRepose),
					},
				},
			})

			time.Sleep(delay)

			response2 := makeRequest(t, http.DefaultClient, appBuilder.URI())

			assert.Equal(t, otherExpectedRepose, response2)
		})

		t.Run("HTTPS", func(t *testing.T) {
			httpClient := testutils.SetupHTTPSTest(t, fs) // nolint: staticcheck
			port := freeport.GetPort()
			appBuilder := appbuilder.NewAppBuilder(t).
				WithFs(fs).
				WithHTTPS()

			uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPSPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(expectedResponse),
					},
				},
			})

			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, httpClient, appBuilder.URI())

			assert.Equal(t, expectedResponse, response)

			uncorsApp.Restart(ctx, &config.UncorsConfig{
				Mappings: config.Mappings{
					config.Mapping{
						From:  hosts.Loopback.HTTPSPort(port),
						To:    hosts.Github.HTTPS(),
						Mocks: mocks(otherExpectedRepose),
					},
				},
			})

			time.Sleep(delay)

			response2 := makeRequest(t, httpClient, appBuilder.URI())

			assert.Equal(t, otherExpectedRepose, response2)
		})
	}))
}

func makeRequest(t *testing.T, httpClient *http.Client, uri *url.URL) string {
	t.Helper()

	res, err := httpClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
	testutils.CheckNoError(t, err)

	defer helpers.CloseSafe(res.Body)

	data, err := io.ReadAll(res.Body)
	testutils.CheckNoError(t, err)

	return string(data)
}

func mocks(response string) config.Mocks {
	return config.Mocks{
		config.Mock{
			Matcher: config.RequestMatcher{
				Path: "/",
			},
			Response: config.Response{
				Code: http.StatusOK,
				Raw:  response,
			},
		},
	}
}

func TestApp_Wait(t *testing.T) {
	t.Skip()
	ctx := t.Context()
	fs := afero.NewMemMapFs()

	t.Run("wait for servers to finish", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("test"),
				},
			},
		})

		// Test Wait in a goroutine
		done := make(chan bool)

		go func() {
			uncorsApp.Wait()

			done <- true
		}()

		// Close the app
		err := uncorsApp.Close()
		testutils.CheckNoServerError(t, err)

		// Wait should return after close
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Wait() did not return after Close()")
		}
	}))
}

func TestApp_Shutdown(t *testing.T) {
	t.Skip()
	ctx := t.Context()
	fs := afero.NewMemMapFs()

	t.Run("graceful shutdown", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("test"),
				},
			},
		})

		time.Sleep(delay)

		// Test graceful shutdown
		err := uncorsApp.Shutdown(ctx)
		assert.NoError(t, err)
	}))

	t.Run("shutdown with no servers", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("test"),
				},
			},
		})

		time.Sleep(delay)

		// Shutdown once
		err := uncorsApp.Shutdown(ctx)
		require.NoError(t, err)

		// Shutdown again (should return no error)
		err = uncorsApp.Shutdown(ctx)
		require.NoError(t, err)
	}))
}
