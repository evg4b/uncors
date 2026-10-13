package har

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterReportsOneNoticePerFailureStreak(t *testing.T) {
	// A file where the HAR directory should be makes every flush fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	block := func() { require.NoError(t, os.WriteFile(blocker, nil, 0o600)) }

	block()

	publisher := &testutils.EventRecorder{}
	path := filepath.Join(blocker, "out.har")
	writer := NewWriter(path, WithPublisher(publisher))

	// No entry is added, so the writer goroutine stays idle and flush can be
	// driven from the test.
	writer.flush()
	writer.flush()

	notices := publisher.Notices()
	require.Len(t, notices, 1)
	assert.Equal(t, events.NoticeError, notices[0].Level)
	assert.Contains(t, notices[0].Text, "HAR: cannot write "+path)

	require.NoError(t, os.Remove(blocker))
	writer.flush()
	assert.FileExists(t, path)
	assert.Len(t, publisher.Notices(), 1)

	require.NoError(t, os.RemoveAll(blocker))
	block()
	writer.flush()
	assert.Len(t, publisher.Notices(), 2)

	require.NoError(t, writer.Close())
	assert.Len(t, publisher.Notices(), 2, "the failing flush on Close continues the streak")
}
