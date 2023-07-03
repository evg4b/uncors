package mock

import (
	"fmt"
	"net/http"
	"os"
)

func (h *Handler) serveFileContent(writer http.ResponseWriter, request *http.Request) error {
	fileName := h.response.File
	file, err := h.fs.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
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
