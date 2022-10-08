// nolint: goerr113
package server_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestServerListenAndServe(t *testing.T) {
	t.Run("should return no errors where http server closed", func(t *testing.T) {
		httpMock := mocks.NewListenerMock(t).
			ListenAndServeMock.Expect().Return(http.ErrServerClosed).
			ShutdownMock.Expect(context.Background()).Return(nil)

		instance := server.NewServer(
			server.WithHTTPListener(httpMock),
		)

		err := instance.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})

	t.Run("should return no errors where https server closed", func(t *testing.T) {
		httpMock := mocks.NewListenerMock(t).
			ListenAndServeMock.Return(nil).
			ShutdownMock.Return(nil)

		httpsMock := mocks.NewListenerMock(t).
			ListenAndServeTLSMock.Expect("cert.pem", "key.pem").Return(http.ErrServerClosed).
			ShutdownMock.Return(nil)

		instance := server.NewServer(
			server.WithHTTPListener(httpMock),
			server.WithHTTPSListener(httpsMock),
			server.WithSslCert("cert.pem"),
			server.WithSslKey("key.pem"),
		)

		err := instance.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})

	t.Run("should return error where http shutdown return error", func(t *testing.T) {
		testError := errors.New("test error")

		httpMock := mocks.NewListenerMock(t).
			ListenAndServeMock.Return(nil).
			ShutdownMock.Return(testError)

		httpsMock := mocks.NewListenerMock(t).
			ListenAndServeTLSMock.Expect("cert.pem", "key.pem").Return(http.ErrServerClosed).
			ShutdownMock.Return(testError)

		instance := server.NewServer(
			server.WithHTTPListener(httpMock),
			server.WithHTTPSListener(httpsMock),
			server.WithSslCert("cert.pem"),
			server.WithSslKey("key.pem"),
		)

		err := instance.ListenAndServe(context.Background())

		assert.Error(t, err)
	})

	t.Run("should return run https server where cert and key are not configured", func(t *testing.T) {
		instance := server.NewServer(
			server.WithHTTPListener(
				mocks.NewListenerMock(t).
					ListenAndServeMock.Return(http.ErrServerClosed).
					ShutdownMock.Return(nil),
			),
			server.WithHTTPSListener(mocks.NewListenerMock(t)),
		)

		err := instance.ListenAndServe(context.Background())

		assert.NoError(t, err)
	})
}
