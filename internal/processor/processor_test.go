package processor_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRequestProcessor_HandleRequest(t *testing.T) {
	t.Run("should have correct calling order", func(t *testing.T) {
		tracker := testutils.NewMidelwaresTracker(t)

		requetProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMidelware("middleware1")),
			processor.WithMiddleware(tracker.MakeMidelware("middleware2")),
			processor.WithMiddleware(tracker.MakeFinalMidelware("middleware3")),
		)

		req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()
		http.HandlerFunc(requetProcessor.HandleRequest).
			ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, []string{"middleware1", "middleware2", "middleware3"}, tracker.CallsOrder)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should skip midelwares where next not called", func(t *testing.T) {
		tracker := testutils.NewMidelwaresTracker(t)

		requetProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(tracker.MakeMidelware("middleware1")),
			processor.WithMiddleware(tracker.MakeFinalMidelware("middleware2")),
			processor.WithMiddleware(tracker.MakeMidelware("middleware3")),
		)

		req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()
		http.HandlerFunc(requetProcessor.HandleRequest).
			ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, []string{"middleware1", "middleware2"}, tracker.CallsOrder)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("should send error to response from midelware", func(t *testing.T) {
		expectedErr := errors.New("Test error") // nolint: goerr113
		requetProcessor := processor.NewRequestProcessor(
			processor.WithMiddleware(
				mocks.NewHandlingMiddlewareMock(t).WrapMock.
					Return(func(w http.ResponseWriter, r *http.Request) error {
						return expectedErr
					}),
			),
		)

		req, err := http.NewRequestWithContext(context.TODO(), "POST", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		recorder := httptest.NewRecorder()
		http.HandlerFunc(requetProcessor.HandleRequest).
			ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), expectedErr.Error())
	})
}
