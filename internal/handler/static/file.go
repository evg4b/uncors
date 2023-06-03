package static

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/afero"
)

var errNorHandled = errors.New("request is not handled")

func (m *Middleware) openFile(filePath string) (afero.File, os.FileInfo, error) {
	file, err := m.fs.Open(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("filed to open file: %w", err)
		}

		indexFile, err := m.openIndexFile()
		if err != nil {
			return nil, nil, err
		}

		file = indexFile
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("filed to get information about file: %w", err)
	}

	if stat.IsDir() {
		indexFile, err := m.openIndexFile()
		if err != nil {
			return file, stat, err
		}

		indexFileStat, err := indexFile.Stat()
		if err != nil {
			return file, stat, fmt.Errorf("filed to get information about index file: %w", err)
		}

		file = indexFile
		stat = indexFileStat
	}

	return file, stat, nil
}

func (m *Middleware) openIndexFile() (afero.File, error) {
	if len(m.index) == 0 {
		return nil, errNorHandled
	}

	file, err := m.fs.Open(m.index)
	if err != nil {
		return nil, fmt.Errorf("filed to opend index file: %w", err)
	}

	return file, nil
}
