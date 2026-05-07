package uncorsapp

import (
	"net/url"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recv(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel message")

		return ""
	}
}

func newTestOutput() (*tuiOutput, <-chan string) {
	outputCh := make(chan string, 10)

	return newTuiOutput(outputCh), outputCh
}

func TestTuiOutput_Info(t *testing.T) {
	t.Run("Info sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Info("hello info")
		assert.Contains(t, recv(t, ch), "hello info")
	})

	t.Run("Infof formats and sends message", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Infof("value is %d", 42)
		assert.Contains(t, recv(t, ch), "42")
	})

	t.Run("InfoBox sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.InfoBox("box line one", "box line two")

		msg := recv(t, ch)
		assert.Contains(t, msg, "box line one")
		assert.Contains(t, msg, "box line two")
	})
}

func TestTuiOutput_Error(t *testing.T) {
	t.Run("Error sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Error("something failed")
		assert.Contains(t, recv(t, ch), "something failed")
	})

	t.Run("Errorf formats and sends message", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Errorf("error code %d", 500)
		assert.Contains(t, recv(t, ch), "500")
	})

	t.Run("ErrorBox sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.ErrorBox("err a", "err b")

		msg := recv(t, ch)
		assert.Contains(t, msg, "err a")
		assert.Contains(t, msg, "err b")
	})
}

func TestTuiOutput_Warn(t *testing.T) {
	t.Run("Warn sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Warn("watch out")
		assert.Contains(t, recv(t, ch), "watch out")
	})

	t.Run("Warnf formats and sends message", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Warnf("threshold %d%%", 90)
		assert.Contains(t, recv(t, ch), "90")
	})

	t.Run("WarnBox sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.WarnBox("warn x", "warn y")

		msg := recv(t, ch)
		assert.Contains(t, msg, "warn x")
		assert.Contains(t, msg, "warn y")
	})
}

func TestTuiOutput_Print(t *testing.T) {
	t.Run("Print sends message to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Print("plain text")
		assert.Contains(t, recv(t, ch), "plain text")
	})

	t.Run("Printf formats and sends message", func(t *testing.T) {
		out, ch := newTestOutput()
		out.Printf("count=%d", 7)
		assert.Contains(t, recv(t, ch), "7")
	})
}

func TestTuiOutput_Write(t *testing.T) {
	t.Run("Write sends bytes as string to channel", func(t *testing.T) {
		out, ch := newTestOutput()
		n, err := out.Write([]byte("raw bytes"))
		require.NoError(t, err)
		assert.Equal(t, 9, n)
		assert.Contains(t, recv(t, ch), "raw bytes")
	})

	t.Run("Write returns len(p) on success", func(t *testing.T) {
		out, _ := newTestOutput()
		data := []byte("test data 123")
		n, err := out.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
	})

	t.Run("Write with only whitespace does not send", func(t *testing.T) {
		out, ch := newTestOutput()
		_, err := out.Write([]byte("\n\n"))
		require.NoError(t, err)

		select {
		case msg := <-ch:
			t.Fatalf("expected no message, got %q", msg)
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func TestTuiOutput_Request(t *testing.T) {
	t.Run("Request sends formatted request data", func(t *testing.T) {
		out, outputCh := newTestOutput()
		u, _ := url.Parse("http://example.com/api/resource")
		out.Request(&contracts.ReqestData{
			Method: "GET",
			URL:    u,
			Code:   200,
		})

		msg := recv(t, outputCh)
		assert.NotEmpty(t, msg)
	})
}

func TestTuiOutput_NewPrefixOutput(t *testing.T) {
	t.Run("returns output that shares the same channel", func(t *testing.T) {
		out, outputCh := newTestOutput()
		prefixed := out.NewPrefixOutput("[SVC]")
		prefixed.Info("service message")
		assert.NotEmpty(t, recv(t, outputCh))
	})

	t.Run("NewPrefixOutput implements contracts.Output", func(_ *testing.T) {
		out, _ := newTestOutput()

		_ = out.NewPrefixOutput("prefix")
	})
}
