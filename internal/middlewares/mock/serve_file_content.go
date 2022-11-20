package mock

import (
	"fmt"
	"net/http"
	"os"
)

func (handler *internalHandler) serveFileContent(writer http.ResponseWriter, request *http.Request) error {
	fileName := handler.response.File
	file, err := handler.fs.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("filed to opent file %s: %w", fileName, err)
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("filed to recive file infirmation: %w", err)
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}
