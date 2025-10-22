package infra

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"

	"github.com/dustin/go-humanize"
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

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, errorHeader)
	fmt.Fprintln(writer)
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Occurred error: %s\n", err)
	fmt.Fprintln(writer)

	fmt.Fprint(writer, "Stack trace: ")
	fmt.Fprintln(writer, string(debug.Stack()))

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Fprintln(writer, "Memory usage:")
	fmt.Fprintf(writer, "Alloc = %v\n", humanize.Bytes(memStats.Alloc))
	fmt.Fprintf(writer, "TotalAlloc = %v\n", humanize.Bytes(memStats.TotalAlloc))
	fmt.Fprintf(writer, "Sys = %v\n", humanize.Bytes(memStats.Sys))
	fmt.Fprintf(writer, "NumGC = %v\n", memStats.NumGC)
}
