package infra

import (
	"net/http"

	"github.com/evg4b/uncors/internal/helpers"
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
	message := helpers.Sprintf("%d Error", http.StatusInternalServerError)

	helpers.FPrintln(writer)
	helpers.FPrintln(writer, pageHeader(message))
	helpers.FPrintln(writer)
	helpers.FPrintln(writer, helpers.Sprintf("Occurred error: %s", err))
}

func pageHeader(message string) string {
	letters := putils.LettersFromStringWithStyle(message, &style)
	text, err := pterm.DefaultBigText.WithLetters(letters).Srender()
	if err != nil {
		panic(err)
	}

	return text
}
