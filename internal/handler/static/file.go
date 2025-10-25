package static

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/afero"
)

var errNotHandled = errors.New("request is not handled")

func (h *Middleware) openFile(filePath string) (afero.File, os.FileInfo, error) {
	file, err := h.fs.Open(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("failed to open file: %w", err)
		}

		indexFile, err := h.openIndexFile()
		if err != nil {
			return nil, nil, err
		}

		file = indexFile
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("failed to get information about file: %w", err)
	}

	if stat.IsDir() {
		indexFile, err := h.openIndexFile()
		if err != nil {
			return file, stat, err
		}

		indexFileStat, err := indexFile.Stat()
		if err != nil {
			return file, stat, fmt.Errorf("failed to get information about index file: %w", err)
		}

		file = indexFile
		stat = indexFileStat
	}

	return file, stat, nil
}

func (h *Middleware) openIndexFile() (afero.File, error) {
	if len(h.index) == 0 {
		return nil, errNotHandled
	}

	file, err := h.fs.Open(h.index)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}

	return file, nil
}
