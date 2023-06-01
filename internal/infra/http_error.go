package infra

import (
	"net/http"

	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/go-http-utils/headers"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var style = pterm.Style{}

func HTTPError(writer http.ResponseWriter, err error) {
	header := writer.Header()
	header.Set(headers.ContentType, "text/plain; charset=utf-8")
	header.Set(headers.XContentTypeOptions, "nosniff")

	writer.WriteHeader(http.StatusInternalServerError)
	message := sfmt.Sprintf("%d Error", http.StatusInternalServerError)

	sfmt.Fprintln(writer)
	sfmt.Fprintln(writer, pageHeader(message))
	sfmt.Fprintln(writer)
	sfmt.Fprintln(writer, sfmt.Sprintf("Occurred error: %s", err))
}

func pageHeader(message string) string {
	letters := putils.LettersFromStringWithStyle(message, &style)
	text, err := pterm.DefaultBigText.WithLetters(letters).Srender()
	if err != nil {
		panic(err)
	}

	return text
}
