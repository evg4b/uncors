package infrastructure

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-http-utils/headers"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/samber/lo"
)

var style = pterm.Style{}

func HTTPError(writer http.ResponseWriter, err error) {
	header := writer.Header()
	header.Set(headers.ContentType, "text/plain; charset=utf-8")
	header.Set(headers.XContentTypeOptions, "nosniff")

	writer.WriteHeader(http.StatusInternalServerError)
	message := fmt.Sprintf("%d Error", http.StatusInternalServerError)

	writeLine(writer)
	writeLine(writer, pageHeader(message))
	writeLine(writer)
	writeLine(writer, fmt.Sprintf("Occurred error: %s", err))
}

func pageHeader(message string) string {
	letters := putils.LettersFromStringWithStyle(message, &style)
	text, err := pterm.DefaultBigText.WithLetters(letters).Srender()
	if err != nil {
		panic(err)
	}

	return text
}

func writeLine(writer io.Writer, data ...string) {
	if _, err := fmt.Fprintln(writer, lo.ToAnySlice(data)...); err != nil {
		panic(err)
	}
}
