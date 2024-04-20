package infra

import (
	"net/http"
	"runtime"
	"runtime/debug"

	"github.com/dustin/go-humanize"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/go-http-utils/headers"
)

const errorHeader = `
███████  ██████   ██████      ███████ ██████  ██████   ██████  ██████  
██      ██  ████ ██  ████     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████ ██ ██ ██ ██ ██ ██     █████   ██████  ██████  ██    ██ ██████  
     ██ ████  ██ ████  ██     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████  ██████   ██████      ███████ ██   ██ ██   ██  ██████  ██   ██ `

func HTTPError(writer http.ResponseWriter, err error) {
	header := writer.Header()
	header.Set(headers.ContentType, "text/plain; charset=utf-8")
	header.Set(headers.ContentEncoding, "identity")
	header.Set(headers.CacheControl, "no-cache, no-store, max-age=0, must-revalidate")
	header.Set(headers.Pragma, "no-cache")
	header.Set(headers.XContentTypeOptions, "nosniff")

	header.Del(headers.SetCookie)

	writer.WriteHeader(http.StatusInternalServerError)

	helpers.FPrintln(writer)
	helpers.FPrintln(writer, errorHeader)
	helpers.FPrintln(writer)
	helpers.FPrintln(writer)
	helpers.FPrintln(writer, helpers.Sprintf("Occurred error: %s", err))
	helpers.FPrintln(writer)

	helpers.FPrint(writer, "Stack trace: ")
	helpers.FPrintln(writer, string(debug.Stack()))

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	helpers.FPrintln(writer, "Memory usage:")
	helpers.FPrintf(writer, "Alloc = %v\n", humanize.Bytes(memStats.Alloc))
	helpers.FPrintf(writer, "TotalAlloc = %v\n", humanize.Bytes(memStats.TotalAlloc))
	helpers.FPrintf(writer, "Sys = %v\n", humanize.Bytes(memStats.Sys))
	helpers.FPrintf(writer, "NumGC = %v\n", memStats.NumGC)
}
