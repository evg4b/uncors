package mock

import (
	"fmt"
	"net/http"
	"os"
)

func (m *Middleware) serveFileContent(writer http.ResponseWriter, request *http.Request) error {
	fileName := m.response.File
	file, err := m.fs.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("filed to open file %s: %w", fileName, err)
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("filed to recive file infirmation: %w", err)
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}
