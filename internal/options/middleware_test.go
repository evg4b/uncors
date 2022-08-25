package options_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/options"
	"github.com/evg4b/uncors/internal/processor"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestOptionsMiddlewareWrap(t *testing.T) {
	middleware := options.NewOptionsMiddleware()

	testMethods := []struct {
		name   string
		method string
	}{
		{name: "should skip POST requst", method: http.MethodPost},
		{name: "should skip GET requst", method: http.MethodGet},
		{name: "should skip PATCH requst", method: http.MethodPatch},
		{name: "should skip DELETE requst", method: http.MethodDelete},
		{name: "should skip HEAD requst", method: http.MethodHead},
		{name: "should skip PUT requst", method: http.MethodPut},
		{name: "should skip CONNECT requst", method: http.MethodConnect},
		{name: "should skip TRACE requst", method: http.MethodTrace},
	}
	for _, testCase := range testMethods {
		t.Run(testCase.name, func(t *testing.T) {
			tracker := testutils.NewMidelwaresTracker(t)
			proc := processor.NewRequestProcessor(
				processor.WithMiddleware(middleware),
				processor.WithMiddleware(tracker.MakeFinalMidelware("final")),
			)

			req, err := http.NewRequestWithContext(context.TODO(), testCase.method, "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			http.HandlerFunc(proc.HandleRequest).
				ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, []string{"final"}, tracker.CallsOrder)
		})
	}

	t.Run("shoud handle OPTIONS requst", func(t *testing.T) {
		tracker := testutils.NewMidelwaresTracker(t)
		proc := processor.NewRequestProcessor(
			processor.WithMiddleware(middleware),
			processor.WithMiddleware(tracker.MakeFinalMidelware("final")),
		)

		req, err := http.NewRequestWithContext(context.TODO(), http.MethodOptions, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		http.HandlerFunc(proc.HandleRequest).
			ServeHTTP(httptest.NewRecorder(), req)

		assert.Equal(t, []string{}, tracker.CallsOrder)
	})

	t.Run("should correctly create respoce", func(t *testing.T) {
		testMethods := []struct {
			name     string
			headers  http.Header
			expected http.Header
		}{
			{
				name:     "should do not change empty headers",
				headers:  http.Header(map[string][]string{}),
				expected: http.Header(map[string][]string{}),
			},
			{
				name: "should do not skip not access-control-request-* headers",
				headers: http.Header{
					"Host":          {"www.host.com"},
					"Content-Type":  {"application/json"},
					"Authorization": {"Bearer Token"},
				},
				expected: http.Header{},
			},
			{
				name: "should allow all access-control-request-* headers",
				headers: http.Header{
					"Access-Control-Request-Headers": {"X-PINGOTHER, Content-Type"},
					"Access-Control-Request-Method":  {http.MethodPost, http.MethodDelete},
				},
				expected: http.Header{
					"Access-Control-Allow-Headers": {"X-PINGOTHER, Content-Type"},
					"Access-Control-Allow-Method":  {http.MethodPost, http.MethodDelete},
				},
			},
		}
		for _, testCase := range testMethods {
			t.Run(testCase.name, func(t *testing.T) {
				proc := processor.NewRequestProcessor(processor.WithMiddleware(middleware))
				req, err := http.NewRequestWithContext(context.TODO(), http.MethodOptions, "/", nil)
				if err != nil {
					t.Fatal(err)
				}

				req.Header = testCase.headers

				rr := httptest.NewRecorder()
				http.HandlerFunc(proc.HandleRequest).
					ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)
				assert.Equal(t, testCase.expected, rr.Header())
			})
		}
	})
}
