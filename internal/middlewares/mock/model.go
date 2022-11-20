package mock

type Response struct {
	Code       int               `yaml:"code"`
	Headers    map[string]string `yaml:"headers"`
	RawContent string            `yaml:"raw-content"` //nolint:tagliatelle
	File       string            `yaml:"file"`
}

type Mock struct {
	Path     string            `yaml:"path"`
	Method   string            `yaml:"method"`
	Queries  map[string]string `yaml:"queries"`
	Headers  map[string]string `yaml:"headers"`
	Response Response          `yaml:"response"`
}
