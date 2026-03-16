package har

import "time"

// HAR represents the root HTTP Archive object (HAR 1.2).
type HAR struct {
	Log Log `json:"log"`
}

// Log is the top-level container for HAR data.
type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

// Creator identifies the application that produced the HAR.
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entry represents a single request/response pair.
type Entry struct {
	StartedDateTime time.Time `json:"startedDateTime"`
	Time            float64   `json:"time"` // ms
	Request         Request   `json:"request"`
	Response        Response  `json:"response"`
	Timings         Timings   `json:"timings"`
}

// Request describes an HTTP request.
type Request struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []NameValue `json:"headers"`
	QueryString []NameValue `json:"queryString"`
	Cookies     []Cookie    `json:"cookies"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
	PostData    *PostData   `json:"postData,omitempty"`
}

// Response describes an HTTP response.
type Response struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []NameValue `json:"headers"`
	Cookies     []Cookie    `json:"cookies"`
	Content     Content     `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
}

// Content holds response body details.
// When Encoding is "base64", Text contains the base64-encoded body bytes
// (used for payloads that could not be decoded, e.g. unknown compressions).
type Content struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

// Timings breaks down request time into phases (all in ms).
type Timings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// NameValue is a generic key/value pair used for headers and query params.
type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Cookie represents a single HTTP cookie.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PostData holds request body information.
type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}
