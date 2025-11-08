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
			httpClient := testutils.SetupHTTPSTest(t, fs)
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
			httpClient := testutils.SetupHTTPSTest(t, fs)
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

func TestApp_GetListenerAddr(t *testing.T) {
	ctx := t.Context()
	fs := afero.NewMemMapFs()

	t.Run("get HTTP listener address", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
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

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		addr := uncorsApp.GetListenerAddr(port)
		assert.NotNil(t, addr)
		assert.Contains(t, addr.String(), "127.0.0.1")
	}))

	t.Run("get listener for non-existent port", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
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

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		addr := uncorsApp.GetListenerAddr(9999)
		assert.Nil(t, addr)
	}))

	t.Run("get listener after shutdown", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
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

		err := uncorsApp.Shutdown(ctx)
		require.NoError(t, err)

		addr := uncorsApp.GetListenerAddr(port)
		assert.Nil(t, addr)
	}))
}

func TestApp_MultiPort(t *testing.T) {
	ctx := t.Context()
	fs := afero.NewMemMapFs()

	t.Run("multiple HTTP ports", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port1 := freeport.GetPort()
		port2 := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port1),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("response1"),
				},
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port2),
					To:    hosts.Example.HTTPS(),
					Mocks: mocks("response2"),
				},
			},
		})

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		// Verify both ports are listening
		addr1 := uncorsApp.GetListenerAddr(port1)
		assert.NotNil(t, addr1)

		addr2 := uncorsApp.GetListenerAddr(port2)
		assert.NotNil(t, addr2)

		// Verify HTTPAddr returns the first HTTP address
		httpAddr := uncorsApp.HTTPAddr()
		assert.NotNil(t, httpAddr)
	}))

	t.Run("mixed HTTP and HTTPS ports", func(t *testing.T) {
		testutils.SetupHTTPSTest(t, fs)

		httpPort := freeport.GetPort()
		httpsPort := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).
			WithFs(fs).
			WithHTTPS()

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(httpPort),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("http-response"),
				},
				config.Mapping{
					From:  hosts.Loopback.HTTPSPort(httpsPort),
					To:    hosts.Example.HTTPS(),
					Mocks: mocks("https-response"),
				},
			},
		})

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		// Verify both HTTP and HTTPS addresses are available
		httpAddr := uncorsApp.HTTPAddr()
		assert.NotNil(t, httpAddr)

		httpsAddr := uncorsApp.HTTPSAddr()
		assert.NotNil(t, httpsAddr)
	})

	t.Run("HTTPAddr and HTTPSAddr return nil when no servers", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
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

		err := uncorsApp.Shutdown(ctx)
		require.NoError(t, err)

		// After shutdown, addresses should be nil
		httpAddr := uncorsApp.HTTPAddr()
		assert.Nil(t, httpAddr)

		httpsAddr := uncorsApp.HTTPSAddr()
		assert.Nil(t, httpsAddr)
	}))
}

func TestApp_StaticAndCacheHandler(t *testing.T) {
	ctx := t.Context()
	fs := afero.NewMemMapFs()

	err := afero.WriteFile(fs, "/static/test.txt", []byte("static content"), 0o644)
	testutils.CheckNoError(t, err)

	t.Run("static file handler", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From: hosts.Loopback.HTTPPort(port),
					To:   hosts.Github.HTTPS(),
					Statics: []config.StaticDirectory{
						{
							Path:  "/static",
							Dir:   "/static",
							Index: "index.html",
						},
					},
				},
			},
		})

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		// Just verify the app started with static handler
		addr := uncorsApp.GetListenerAddr(port)
		assert.NotNil(t, addr)
	}))

	t.Run("cache handler", testutils.LogTest(func(t *testing.T, _ *bytes.Buffer) {
		port := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			CacheConfig: config.CacheConfig{
				Methods:        []string{http.MethodGet},
				ExpirationTime: 5 * time.Minute,
				ClearTime:      10 * time.Minute,
			},
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(port),
					To:    hosts.Github.HTTPS(),
					Cache: config.CacheGlobs{"/*"},
					Mocks: mocks("cached response"),
				},
			},
		})

		defer func() {
			err := uncorsApp.Close()
			testutils.CheckNoServerError(t, err)
		}()

		time.Sleep(delay)

		// Just verify the app started with cache handler
		addr := uncorsApp.GetListenerAddr(port)
		assert.NotNil(t, addr)
	}))
}

func TestApp_HTTPSWithoutCerts(t *testing.T) {
	t.Skip()

	ctx := t.Context()
	fs := afero.NewMemMapFs()

	t.Run("HTTPS mapping without cert configuration", testutils.LogTest(func(t *testing.T, logBuffer *bytes.Buffer) {
		httpsPort := freeport.GetPort()
		httpPort := freeport.GetPort()
		appBuilder := appbuilder.NewAppBuilder(t).WithFs(fs)

		// Start with both HTTP and HTTPS mappings, but no certs
		// Only HTTP should start successfully
		uncorsApp := appBuilder.Start(ctx, &config.UncorsConfig{
			Mappings: config.Mappings{
				config.Mapping{
					From:  hosts.Loopback.HTTPPort(httpPort),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("http-test"),
				},
				config.Mapping{
					From:  hosts.Loopback.HTTPSPort(httpsPort),
					To:    hosts.Github.HTTPS(),
					Mocks: mocks("https-test"),
				},
			},
		})

		time.Sleep(delay)

		// HTTP server should start
		httpAddr := uncorsApp.HTTPAddr()
		assert.NotNil(t, httpAddr)

		// HTTPS server should not start without certs
		httpsAddr := uncorsApp.HTTPSAddr()
		assert.Nil(t, httpsAddr)

		// Close the app to ensure all goroutines complete before checking logs
		err := uncorsApp.Close()
		testutils.CheckNoServerError(t, err)

		// Check that warning was logged
		assert.Contains(t, logBuffer.String(), "HTTPS mapping")
		assert.Contains(t, logBuffer.String(), "no cert/key configured")
	}))
}
