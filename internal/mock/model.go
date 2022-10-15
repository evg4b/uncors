package mock

type Response struct {
	Code        int
	Headers     map[string]string
	RawContent  string
	ContentFile string
}

type Mock struct {
	Path     string
	Method   string
	Queries  map[string]string
	Headers  map[string]string
	Response Response
}
