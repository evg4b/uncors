package engine_test

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const harPath = "/traffic.har"

func recordingConfiguration(upstream string, harPort int, otherPorts ...int) *config.UncorsConfig {
	mappings := make(config.Mappings, 0, 1+len(otherPorts))
	mappings = append(mappings, config.Mapping{
		From: hosts.Localhost.HTTPPort(harPort),
		To:   hosts.Parse(upstream),
		HAR:  config.HARConfig{File: harPath},
	})

	for _, port := range otherPorts {
		mappings = append(mappings, config.Mapping{
			From: hosts.Localhost.HTTPPort(port),
			To:   hosts.Parse(upstream),
		})
	}

	return &config.UncorsConfig{
		CacheConfig: config.CacheConfig{MaxSize: 100, ExpirationTime: time.Minute},
		Mappings:    config.NormaliseMappings(mappings),
	}
}

func harEntries(t *testing.T, fileSystem afero.Fs) int {
	t.Helper()

	data, err := afero.ReadFile(fileSystem, harPath)
	if err != nil {
		return -1
	}

	var archive har.HAR
	require.NoError(t, json.Unmarshal(data, &archive))

	return len(archive.Log.Entries)
}

func TestEngineDiscardsTheHAROfAGenerationThatNeverServed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "ok")
	}))
	t.Cleanup(upstream.Close)

	t.Run("a failed start does not create the file", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)
		occupy(t, ports[1])

		fileSystem := afero.NewMemMapFs()
		instance := engine.New(fileSystem)

		t.Cleanup(func() { _ = instance.Close() })

		err := instance.Start(t.Context(), recordingConfiguration(upstream.URL, ports[0], ports[1]))
		require.Error(t, err)

		_, err = fileSystem.Stat(harPath)
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("a failed reload keeps the recording of the running generation", func(t *testing.T) {
		ports := testutils.GetFreePorts(t, 2)

		fileSystem := afero.NewMemMapFs()
		instance := engine.New(fileSystem)

		t.Cleanup(func() { _ = instance.Close() })

		require.NoError(t, instance.Start(t.Context(), recordingConfiguration(upstream.URL, ports[0])))

		url := "http://localhost:" + strconv.Itoa(ports[0]) + "/recorded"
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		require.Eventually(t, func() bool { return harEntries(t, fileSystem) == 1 }, 5*time.Second, 50*time.Millisecond)

		occupy(t, ports[1])

		err = instance.Restart(t.Context(), recordingConfiguration(upstream.URL, ports[0], ports[1]))
		require.Error(t, err)

		assert.Equal(t, 1, harEntries(t, fileSystem))
		assertListening(t, ports[0])
	})
}
