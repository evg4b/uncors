package testutils

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/spf13/afero"
)

// TrackingFs wraps an afero.Fs and counts the files opened through it that
// have not been closed yet, so tests can detect leaked file handles.
type TrackingFs struct {
	afero.Fs

	open atomic.Int64
}

func NewTrackingFs(fs afero.Fs) *TrackingFs {
	return &TrackingFs{Fs: fs}
}

// OpenFiles returns the number of files that are still open.
func (f *TrackingFs) OpenFiles() int64 {
	return f.open.Load()
}

func (f *TrackingFs) Open(name string) (afero.File, error) {
	return f.track(f.Fs.Open(name))
}

func (f *TrackingFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return f.track(f.Fs.OpenFile(name, flag, perm))
}

func (f *TrackingFs) Create(name string) (afero.File, error) {
	return f.track(f.Fs.Create(name))
}

func (f *TrackingFs) track(file afero.File, err error) (afero.File, error) {
	if err != nil {
		return file, err
	}

	f.open.Add(1)

	return &trackedFile{File: file, fs: f}, nil
}

type trackedFile struct {
	afero.File

	fs   *TrackingFs
	once sync.Once
}

func (t *trackedFile) Close() error {
	t.once.Do(func() { t.fs.open.Add(-1) })

	return t.File.Close()
}
