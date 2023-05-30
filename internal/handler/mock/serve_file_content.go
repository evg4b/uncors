package mock

import (
	"net/http"
	"os"

	"github.com/evg4b/uncors/internal/sfmt"
)

func (m *Middleware) serveFileContent(writer http.ResponseWriter, request *http.Request) error {
	fileName := m.response.File
	file, err := m.fs.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return sfmt.Errorf("filed to opent file %s: %w", fileName, err)
	}

	stat, err := file.Stat()
	if err != nil {
		return sfmt.Errorf("filed to recive file infirmation: %w", err)
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}
