package static

import (
	"net/http"
	"time"

	"github.com/go-http-utils/headers"
)

var unixEpochTime = time.Unix(0, 0)

// isZeroTime reports whether t is obviously unspecified (either zero or Unix()=0).
func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

// localRedirect gives a Moved Permanently response.
// It does not convert relative paths to absolute paths like Redirect does.
func localRedirect(writer http.ResponseWriter, request *http.Request, newPath string) {
	if request.URL.RawQuery != "" {
		newPath += "?" + request.URL.RawQuery
	}

	writer.Header().Set(headers.Location, newPath)
	writer.WriteHeader(http.StatusMovedPermanently)
}

func setLastModified(w http.ResponseWriter, modificationTime time.Time) {
	if !isZeroTime(modificationTime) {
		timeString := modificationTime.UTC().Format(http.TimeFormat)
		w.Header().Set(headers.LastModified, timeString)
	}
}

func checkIfModifiedSince(r *http.Request, modificationTime time.Time) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	modifiedSince := r.Header.Get(headers.IfModifiedSince)
	if modifiedSince == "" || isZeroTime(modificationTime) {
		return false
	}

	parsedTime, err := http.ParseTime(modifiedSince)
	if err != nil {
		return false
	}

	modificationTime = modificationTime.Truncate(time.Second)

	return modificationTime.After(parsedTime) && !modificationTime.Equal(parsedTime)
}

func writeNotModified(writer http.ResponseWriter) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	header := writer.Header()
	delete(header, headers.ContentType)
	delete(header, headers.ContentLength)
	delete(header, headers.ContentEncoding)
	if header.Get(headers.ETag) != "" {
		delete(header, headers.LastModified)
	}
	writer.WriteHeader(http.StatusNotModified)
}
