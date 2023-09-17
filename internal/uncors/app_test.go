package uncors_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/phayes/freeport"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const delay = 10 * time.Millisecond

func TestUncorsApp(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewOsFs()
	expectedResponse := "UNCORS OK!"

	t.Run("handle request", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		t.Run("HTTP", func(t *testing.T) {
			uncorsApp, uri := createApp(ctx, t, fs, false, &config.UncorsConfig{
				HTTPPort: freeport.GetPort(),
				Mappings: config.Mappings{
					config.Mapping{
						From:  "http://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(expectedResponse),
					},
				},
			})
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, http.DefaultClient, uri)

			assert.Equal(t, expectedResponse, response)
		})

		t.Run("HTTPS", testutils.WithTmpCerts(fs, func(t *testing.T, certs *testutils.Certs) {
			uncorsApp, uri := createApp(ctx, t, fs, true, &config.UncorsConfig{
				HTTPSPort: freeport.GetPort(),
				CertFile:  certs.CertPath,
				KeyFile:   certs.KeyPath,
				Mappings: config.Mappings{
					config.Mapping{
						From:  "https://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(expectedResponse),
					},
				},
			})
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			httpClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: certs.ClientTLSConf,
				},
			}

			response := makeRequest(t, httpClient, uri)

			assert.Equal(t, expectedResponse, response)
		}))
	}))

	t.Run("restart server", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		const otherExpectedRepose = `{ "bla": true }`

		t.Run("HTTP", func(t *testing.T) {
			port := freeport.GetPort()
			uncorsApp, uri := createApp(ctx, t, fs, false, &config.UncorsConfig{
				HTTPPort: port,
				Mappings: config.Mappings{
					config.Mapping{
						From:  "http://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(expectedResponse),
					},
				},
			})
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			response := makeRequest(t, http.DefaultClient, uri)
			assert.Equal(t, expectedResponse, response)

			uncorsApp.Restart(ctx, &config.UncorsConfig{
				HTTPPort: port,
				Mappings: config.Mappings{
					config.Mapping{
						From:  "https://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(otherExpectedRepose),
					},
				},
			})

			time.Sleep(delay)

			response2 := makeRequest(t, http.DefaultClient, uri)

			assert.Equal(t, otherExpectedRepose, response2)
		})

		t.Run("HTTPS", testutils.WithTmpCerts(fs, func(t *testing.T, certs *testutils.Certs) {
			port := freeport.GetPort()
			uncorsApp, uri := createApp(ctx, t, fs, true, &config.UncorsConfig{
				HTTPSPort: port,
				CertFile:  certs.CertPath,
				KeyFile:   certs.KeyPath,
				Mappings: config.Mappings{
					config.Mapping{
						From:  "https://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(expectedResponse),
					},
				},
			})
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			httpClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: certs.ClientTLSConf,
				},
			}

			response := makeRequest(t, httpClient, uri)

			assert.Equal(t, expectedResponse, response)

			uncorsApp.Restart(ctx, &config.UncorsConfig{
				HTTPSPort: port,
				CertFile:  certs.CertPath,
				KeyFile:   certs.KeyPath,
				Mappings: config.Mappings{
					config.Mapping{
						From:  "https://127.0.0.1",
						To:    "https://github.com",
						Mocks: mocks(otherExpectedRepose),
					},
				},
			})

			time.Sleep(delay)

			response2 := makeRequest(t, httpClient, uri)

			assert.Equal(t, otherExpectedRepose, response2)
		}))
	}))
}

func makeRequest(t *testing.T, httpClient *http.Client, uri *url.URL) string {
	res, err := httpClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
	testutils.CheckNoError(t, err)
	defer helpers.CloseSafe(res.Body)

	data, err := io.ReadAll(res.Body)
	testutils.CheckNoError(t, err)

	return string(data)
}

func createApp(
	ctx context.Context,
	t *testing.T, fs afero.Fs, https bool, config *config.UncorsConfig,
) (*uncors.App, *url.URL) {
	app := uncors.CreateApp(fs, "x.x.x")

	go app.Start(ctx, config)

	time.Sleep(delay)

	prefix := "http://"
	if https {
		prefix = "https://"
	}
	addr := app.HTTPAddr().String()
	if https {
		addr = app.HTTPSAddr().String()
	}
	uri, err := url.Parse(prefix + addr)
	testutils.CheckNoError(t, err)

	return app, uri
}

func mocks(response string) config.Mocks {
	return config.Mocks{
		config.Mock{
			Path: "/",
			Response: config.Response{
				Code: http.StatusOK,
				Raw:  response,
			},
		},
	}
}
