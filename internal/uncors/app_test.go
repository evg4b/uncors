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

const delay = 50 * time.Millisecond

func TestUncorsApp(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewOsFs()
	expectedResponse := "UNCORS OK!"

	mocks := config.Mocks{
		config.Mock{
			Path: "/",
			Response: config.Response{
				Code: http.StatusOK,
				Raw:  expectedResponse,
			},
		},
	}

	t.Run("handle request", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		t.Run("HTTP", func(t *testing.T) {
			uncorsApp := uncors.CreateApp(fs, "x.x.x")
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			go func() {
				uncorsApp.Start(ctx, &config.UncorsConfig{
					HTTPPort: freeport.GetPort(),
					Mappings: config.Mappings{
						config.Mapping{
							From:  "http://127.0.0.1",
							To:    "https://github.com",
							Mocks: mocks,
						},
					},
				})
			}()

			time.Sleep(delay)
			uri, err := url.Parse("http://" + uncorsApp.HTTPAddr().String())
			testutils.CheckNoError(t, err)

			res, err := http.DefaultClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
			testutils.CheckNoError(t, err)
			defer helpers.CloseSafe(res.Body)

			data, err := io.ReadAll(res.Body)
			testutils.CheckNoError(t, err)

			assert.Equal(t, expectedResponse, string(data))
		})

		t.Run("HTTPS", testutils.WithTmpCerts(fs, func(t *testing.T, certs *testutils.Certs) {
			uncorsApp := uncors.CreateApp(fs, "x.x.x")
			defer func() {
				err := uncorsApp.Close()
				testutils.CheckNoServerError(t, err)
			}()

			go func() {
				uncorsApp.Start(ctx, &config.UncorsConfig{
					HTTPSPort: freeport.GetPort(),
					CertFile:  certs.CertPath,
					KeyFile:   certs.KeyPath,
					Mappings: config.Mappings{
						config.Mapping{
							From:  "https://127.0.0.1",
							To:    "https://github.com",
							Mocks: mocks,
						},
					},
				})
			}()

			time.Sleep(delay)
			uri, err := url.Parse("https://" + uncorsApp.HTTPSAddr().String())
			testutils.CheckNoError(t, err)

			httpClient := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: certs.ClientTLSConf,
				},
			}

			res, err := httpClient.Do(&http.Request{URL: uri, Method: http.MethodGet})
			testutils.CheckNoError(t, err)
			defer helpers.CloseSafe(res.Body)

			data, err := io.ReadAll(res.Body)
			testutils.CheckNoError(t, err)

			assert.Equal(t, expectedResponse, string(data))
		}))
	}))
}
