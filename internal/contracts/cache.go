package contracts

type CachedHeader struct {
	Name  string
	Value []string
}

type CachedResponse struct {
	Code    int
	Body    []byte
	Headers []CachedHeader
}

type Cache interface {
	Get(key string) (CachedResponse, bool)
	Set(key string, value CachedResponse)
}
