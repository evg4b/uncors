package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestServerListenAndServe(t *testing.T) {
	t.Run("should return no errors where http server closeed", func(t *testing.T) {
		httpMock := mocks.NewListnerMock(t).
			ListenAndServeMock.Expect().Return(http.ErrServerClosed).
			ShutdownMock.Expect(context.Background()).Return(nil)

		server := server.NewServer(
			server.WithHTTPListner(httpMock),
		)

		err := server.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})

	t.Run("should return no errors where https server closeed", func(t *testing.T) {
		httpMock := mocks.NewListnerMock(t).
			ListenAndServeMock.Return(nil).
			ShutdownMock.Return(nil)

		httpsMock := mocks.NewListnerMock(t).
			ListenAndServeTLSMock.Expect("cert.pem", "key.pem").Return(http.ErrServerClosed).
			ShutdownMock.Return(nil)

		server := server.NewServer(
			server.WithHTTPListner(httpMock),
			server.WithHTTPSListner(httpsMock),
			server.WithSslCert("cert.pem"),
			server.WithSslKey("key.pem"),
		)

		err := server.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})

	t.Run("should return run https server where cert and key are not configured", func(t *testing.T) {
		server := server.NewServer(
			server.WithHTTPListner(
				mocks.NewListnerMock(t).
					ListenAndServeMock.Return(http.ErrServerClosed).
					ShutdownMock.Return(nil),
			),
			server.WithHTTPSListner(mocks.NewListnerMock(t)),
		)

		err := server.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})
}
