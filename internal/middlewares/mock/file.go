package mock

import (
	"net/http"
	"os"
)

func (handler *internalHandler) serveFile(writer http.ResponseWriter, request *http.Request) error {
	file, err := handler.fs.OpenFile(handler.response.File, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}
