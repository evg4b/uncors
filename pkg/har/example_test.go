package har_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/evg4b/uncors/pkg/har"
)

func ExampleEncode() {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/search?q=har", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		Header:     http.Header{"Content-Type": {"text/plain"}},
	}

	entry := har.NewEntry(
		time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC),
		25*time.Millisecond,
		har.NewRequest(req, nil),
		har.NewResponse(resp, []byte("found")),
	)

	_ = har.Encode(os.Stdout, har.New(har.Creator{Name: "example", Version: "1.0"}, entry))
	// Output:
	// {
	//   "log": {
	//     "version": "1.2",
	//     "creator": {
	//       "name": "example",
	//       "version": "1.0"
	//     },
	//     "entries": [
	//       {
	//         "startedDateTime": "2026-10-16T12:00:00Z",
	//         "time": 25,
	//         "request": {
	//           "method": "GET",
	//           "url": "http://example.com/search?q=har",
	//           "httpVersion": "HTTP/1.1",
	//           "cookies": [],
	//           "headers": [],
	//           "queryString": [
	//             {
	//               "name": "q",
	//               "value": "har"
	//             }
	//           ],
	//           "headersSize": -1,
	//           "bodySize": 0
	//         },
	//         "response": {
	//           "status": 200,
	//           "statusText": "OK",
	//           "httpVersion": "HTTP/1.1",
	//           "cookies": [],
	//           "headers": [
	//             {
	//               "name": "Content-Type",
	//               "value": "text/plain"
	//             }
	//           ],
	//           "content": {
	//             "size": 5,
	//             "mimeType": "text/plain",
	//             "text": "found"
	//           },
	//           "redirectURL": "",
	//           "headersSize": -1,
	//           "bodySize": 5
	//         },
	//         "cache": {},
	//         "timings": {
	//           "send": 0,
	//           "wait": 25,
	//           "receive": 0
	//         }
	//       }
	//     ]
	//   }
	// }
}

func ExampleNewWriter() {
	path := filepath.Join(os.TempDir(), "example.har")
	defer os.Remove(path)

	writer := har.NewWriter(path, har.WithCreator(har.Creator{Name: "example", Version: "1.0"}))
	writer.AddEntry(har.Entry{Request: har.Request{Method: http.MethodGet, URL: "http://example.com/"}})
	_ = writer.Close()

	file, _ := os.Open(path)
	defer file.Close()

	archive, _ := har.Decode(file)
	os.Stdout.WriteString(archive.Log.Entries[0].Request.URL + "\n")
	// Output: http://example.com/
}
