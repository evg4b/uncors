package framework

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TestCase represents a complete integration test case.
type TestCase struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Backend     BackendConfig    `yaml:"backend"`
	Uncors      UncorsConfig     `yaml:"uncors"`
	Tests       []TestDefinition `yaml:"tests"`
}

// BackendConfig defines the mock backend server configuration.
type BackendConfig struct {
	Port      int              `yaml:"port"`
	TLS       bool             `yaml:"tls"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

// UncorsConfig defines the uncors proxy configuration.
type UncorsConfig struct {
	HTTPPort  int             `yaml:"http-port"`
	HTTPSPort int             `yaml:"https-port"`
	Config    map[string]any  `yaml:"config"`
	ConfigRaw string          `yaml:"config-raw"`
	Mappings  []MappingConfig `yaml:"mappings"`
}

// MappingConfig represents a single mapping in uncors configuration.
type MappingConfig struct {
	From     string          `yaml:"from"`
	To       string          `yaml:"to"`
	Mocks    []MockConfig    `yaml:"mocks"`
	Statics  []StaticConfig  `yaml:"statics"`
	Cache    []string        `yaml:"cache"`
	Rewrites []RewriteConfig `yaml:"rewrites"`
}

// MockConfig represents a mock endpoint in uncors configuration.
type MockConfig struct {
	Path     string             `yaml:"path"`
	Response MockResponseConfig `yaml:"response"`
}

// MockResponseConfig defines the mock response in uncors format.
type MockResponseConfig struct {
	Code    int               `yaml:"code"`
	Raw     string            `yaml:"raw,omitempty"`
	File    string            `yaml:"file,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// StaticConfig represents static file serving configuration.
type StaticConfig struct {
	Path  string `yaml:"path"`
	Dir   string `yaml:"dir"`
	Index string `yaml:"index,omitempty"`
}

// RewriteConfig represents URL rewriting configuration.
type RewriteConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// TestDefinition represents a single test within a test case.
type TestDefinition struct {
	Name     string            `yaml:"name"`
	Request  RequestConfig     `yaml:"request"`
	Expected ExpectedResponse  `yaml:"expected"`
	DNS      map[string]string `yaml:"dns"`
}

// RequestConfig defines the HTTP request to be made.
type RequestConfig struct {
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

// ExpectedResponse defines the expected HTTP response.
type ExpectedResponse struct {
	Status       int                    `yaml:"status"`
	Body         string                 `yaml:"body"`
	BodyContains []string               `yaml:"body-contains"`
	BodyJSON     map[string]interface{} `yaml:"body-json"`
	Headers      map[string]string      `yaml:"headers"`
	HeadersExist []string               `yaml:"headers-exist"`
}

// LoadTestCase loads a test case from a YAML file.
func LoadTestCase(path string) (*TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var testCase TestCase
	if err := yaml.Unmarshal(data, &testCase); err != nil {
		return nil, err
	}

	return &testCase, nil
}

// LoadTestCasesFromDir loads all test cases from a directory.
func LoadTestCasesFromDir(dir string) ([]*TestCase, error) {
	var testCases []*TestCase

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if ext := filepath.Ext(entry.Name()); ext == ".yaml" || ext == ".yml" {
			path := filepath.Join(dir, entry.Name())
			testCase, err := LoadTestCase(path)
			if err != nil {
				return nil, err
			}
			testCases = append(testCases, testCase)
		}
	}

	return testCases, nil
}
