package processor_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRequestProcessorHandleRequest(t *testing.T) {
	t.Run("should have correct calling order", func(t *testing.T) {
		tracker := mocks.NewMiddlewaresTracker(t)

		requestProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMiddleware("middleware1")),
			processor.WithMiddleware(tracker.MakeMiddleware("middleware2")),
			processor.WithMiddleware(tracker.MakeFinalMiddleware("middleware3")),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		recorder := httptest.NewRecorder()
		requestProcessor.ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, []string{"middleware1", "middleware2", "middleware3"}, tracker.CallsOrder)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should skip middlewares where next not called", func(t *testing.T) {
		tracker := mocks.NewMiddlewaresTracker(t)

		requestProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMiddleware("middleware1")),
			processor.WithMiddleware(tracker.MakeFinalMiddleware("middleware2")),
			processor.WithMiddleware(tracker.MakeMiddleware("middleware3")),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		recorder := httptest.NewRecorder()
		requestProcessor.ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, []string{"middleware1", "middleware2"}, tracker.CallsOrder)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should send error to response from middleware", func(t *testing.T) {
		requestProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(
				mocks.NewHandlingMiddlewareMock(t).WrapMock.
					Return(func(w http.ResponseWriter, r *http.Request) error {
						return assert.AnError
					}),
			),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		recorder := httptest.NewRecorder()
		requestProcessor.ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		testutils.CheckNoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), assert.AnError.Error())
	})

	t.Run("should provide correct scheme", func(t *testing.T) {
		t.Run("http", func(t *testing.T) {
			requestProcessor := processor.NewRequestProcessor(
				processor.WithMiddleware(
					mocks.NewHandlingMiddlewareMock(t).WrapMock.
						Return(func(w http.ResponseWriter, r *http.Request) error {
							assert.Equal(t, "http", r.URL.Scheme)

							return nil
						}),
				),
			)

			req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
			testutils.CheckNoError(t, err)

			requestProcessor.ServeHTTP(httptest.NewRecorder(), req)
		})

		t.Run("https", func(t *testing.T) {
			requestProcessor := processor.NewRequestProcessor(
				processor.WithMiddleware(
					mocks.NewHandlingMiddlewareMock(t).WrapMock.
						Return(func(w http.ResponseWriter, r *http.Request) error {
							assert.Equal(t, "https", r.URL.Scheme)

							return nil
						}),
				),
			)

			req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
			testutils.CheckNoError(t, err)
			req.TLS = &tls.ConnectionState{}

			requestProcessor.ServeHTTP(httptest.NewRecorder(), req)
		})
	})

	t.Run("call default handler by default", func(t *testing.T) {
		requestProcessor := processor.NewRequestProcessor()

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/", nil)
		testutils.CheckNoError(t, err)

		recorder := httptest.NewRecorder()
		requestProcessor.ServeHTTP(recorder, req)

		assert.Equal(t, "UNCORS error: failed request handler\n", recorder.Body.String())
	})
}
