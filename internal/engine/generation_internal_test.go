package engine

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	harlib "github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordingTo(file string) *config.UncorsConfig {
	return &config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings: config.Mappings{{
			From: hosts.Localhost.HTTP(),
			To:   hosts.Localhost.HTTPS(),
			HAR:  config.HARConfig{File: file},
		}},
	}
}

func TestDiscardReplacedHAR(t *testing.T) {
	tests := []struct {
		name    string
		next    string
		written bool
	}{
		{name: "discards the writer of a file the next generation records to", next: "/session.har"},
		{name: "keeps the writer of a file the next generation does not record to", next: "/other.har", written: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			proxy := New(fs)

			previous, err := proxy.BuildGeneration(recordingTo("/session.har"))
			require.NoError(t, err)

			next, err := proxy.BuildGeneration(recordingTo(testCase.next))
			require.NoError(t, err)

			require.Len(t, previous.harWriters, 1)
			require.True(t, previous.harWriters[0].AddEntry(harlib.Entry{Request: harlib.Request{Method: http.MethodGet}}))

			previous.discardReplacedHAR(next)
			require.NoError(t, previous.Close())
			require.NoError(t, next.Discard())

			exists, err := afero.Exists(fs, "/session.har")
			require.NoError(t, err)
			assert.Equal(t, testCase.written, exists)
		})
	}
}
