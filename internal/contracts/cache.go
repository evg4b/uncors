package contracts

import "time"

type CachedHeader struct {
	Name  string
	Value []string
}

type CachedResponse struct {
	Code    int
	Body    []byte
	Headers []CachedHeader
	Vary    []string
	Stored  time.Time
}

type Cache interface {
	Get(key string) (CachedResponse, bool)
	Set(key string, value CachedResponse) bool
}
