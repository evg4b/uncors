package server_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRequestPrinter(t *testing.T) {
	t.Run("outputs request when Done=true and Data is not nil", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)

		data := &contracts.RequestData{
			Method: "GET",
			Code:   200,
		}

		output.RequestMock.Set(func(_ *contracts.RequestData) {})

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{
			ID:        1,
			Method:    "GET",
			URL:       &url.URL{Path: "/test"},
			StartedAt: time.Now(),
			Done:      true,
			Data:      data,
		})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.NotEmpty(t, output.RequestMock.Calls(), "Request not called")
	})

	t.Run("skips request when Done=false", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)

		data := &contracts.RequestData{
			Method: "GET",
			Code:   200,
		}

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{
			ID:        1,
			Method:    "GET",
			URL:       &url.URL{Path: "/test"},
			StartedAt: time.Now(),
			Done:      false,
			Data:      data,
		})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.Empty(t, output.RequestMock.Calls(), "Request should not be called when Done=false")
	})

	t.Run("skips request when Data is nil", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{
			ID:        1,
			Method:    "GET",
			URL:       &url.URL{Path: "/test"},
			StartedAt: time.Now(),
			Done:      true,
			Data:      nil,
		})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.Empty(t, output.RequestMock.Calls(), "Request should not be called when Data is nil")
	})

	t.Run("uses NewPrefixOutput when Prefix is set", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)
		prefixedOutput := mocks.NewOutputMock(t)

		const prefix = "PROXY"

		data := &contracts.RequestData{
			Method: "GET",
			Code:   200,
		}

		output.NewPrefixOutputMock.Set(func(_ string) contracts.Output {
			return prefixedOutput
		})
		prefixedOutput.RequestMock.Set(func(_ *contracts.RequestData) {})

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{
			ID:        1,
			Method:    "GET",
			URL:       &url.URL{Path: "/test"},
			StartedAt: time.Now(),
			Prefix:    prefix,
			Done:      true,
			Data:      data,
		})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.NotEmpty(t, output.NewPrefixOutputMock.Calls(), "NewPrefixOutput not called")
		assert.NotEmpty(t, prefixedOutput.RequestMock.Calls(), "Request on prefixed output not called")
	})

	t.Run("uses direct output when Prefix is empty", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)

		data := &contracts.RequestData{
			Method: "GET",
			Code:   200,
		}

		output.RequestMock.Set(func(_ *contracts.RequestData) {})

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{
			ID:        1,
			Method:    "GET",
			URL:       &url.URL{Path: "/test"},
			StartedAt: time.Now(),
			Prefix:    "",
			Done:      true,
			Data:      data,
		})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.NotEmpty(t, output.RequestMock.Calls(), "Request not called on direct output")
		assert.Empty(t, output.NewPrefixOutputMock.Calls(), "NewPrefixOutput should not be called")
	})

	t.Run("processes multiple events correctly", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)
		prefixedOutput := mocks.NewOutputMock(t)

		data1 := &contracts.RequestData{Method: "GET", Code: 200}
		data2 := &contracts.RequestData{Method: "POST", Code: 201}
		data3 := &contracts.RequestData{Method: "DELETE", Code: 204}

		output.RequestMock.Set(func(_ *contracts.RequestData) {})
		output.NewPrefixOutputMock.Set(func(_ string) contracts.Output {
			return prefixedOutput
		})
		prefixedOutput.RequestMock.Set(func(_ *contracts.RequestData) {})

		go server.RequestPrinter(tracker, output)

		tracker.Emit(server.RequestEvent{ID: 1, Done: true, Data: data1})
		tracker.Emit(server.RequestEvent{ID: 2, Prefix: "MOD1", Done: true, Data: data2})
		tracker.Emit(server.RequestEvent{ID: 3, Done: false, Data: &contracts.RequestData{Method: "PATCH", Code: 200}})
		tracker.Emit(server.RequestEvent{ID: 4, Done: true, Data: data3})

		tracker.Close()

		time.Sleep(10 * time.Millisecond)

		assert.Len(t, output.RequestMock.Calls(), 2, "expected 2 direct Request calls")
		assert.Len(t, output.NewPrefixOutputMock.Calls(), 1, "expected 1 NewPrefixOutput call")
		assert.Len(t, prefixedOutput.RequestMock.Calls(), 1, "expected 1 prefixed Request call")
	})

	t.Run("handles tracker closure gracefully", func(t *testing.T) {
		tracker := server.NewRequestTracker()
		output := mocks.NewOutputMock(t)

		done := make(chan struct{})

		go func() {
			server.RequestPrinter(tracker, output)
			close(done)
		}()

		tracker.Close()

		<-done
	})
}
