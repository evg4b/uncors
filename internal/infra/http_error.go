package infra

import (
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/sfmt"
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
	message := sfmt.Sprintf("%d Error", http.StatusInternalServerError)

	writeLine(writer)
	writeLine(writer, pageHeader(message))
	writeLine(writer)
	writeLine(writer, sfmt.Sprintf("Occurred error: %s", err))
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
	sfmt.Fprintln(writer, lo.ToAnySlice(data)...)
}
