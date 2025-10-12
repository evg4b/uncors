package framework

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Errors.
var (
	ErrUncorsStartTimeout  = errors.New("timeout waiting for uncors to start")
	ErrProjectRootNotFound = errors.New("project root not found")
)

// Constants.
const (
	defaultHTTPPort      = 3000
	yamlIndent           = 2
	uncorsStartupTimeout = 10 * time.Second
	startupCheckInterval = 100 * time.Millisecond
	requestTimeout       = 30 * time.Second
)

// TestRunner orchestrates integration tests.
type TestRunner struct {
	t                *testing.T
	testCase         *TestCase
	backendServer    *TestServer
	uncorsCmd        *exec.Cmd
	uncorsConfigPath string
	httpClient       *http.Client
}

// NewTestRunner creates a new test runner for the given test case.
func NewTestRunner(t *testing.T, testCase *TestCase) *TestRunner {
	return &TestRunner{
		t:        t,
		testCase: testCase,
	}
}

// Setup initializes the test environment.
func (r *TestRunner) Setup() error {
	// Start backend server
	if len(r.testCase.Backend.Endpoints) > 0 {
		if r.testCase.Backend.TLS {
			r.backendServer = NewTestServerTLS(r.testCase.Backend.Endpoints)
		} else {
			r.backendServer = NewTestServer(r.testCase.Backend.Endpoints)
		}
		r.t.Logf("Started backend server at: %s", r.backendServer.URL())
	}

	// Create HTTP client
	r.httpClient = &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Generate uncors configuration
	if err := r.generateUncorsConfig(); err != nil {
		return fmt.Errorf("failed to generate uncors config: %w", err)
	}

	// Start uncors
	if err := r.startUncors(); err != nil {
		return fmt.Errorf("failed to start uncors: %w", err)
	}

	// Wait for uncors to be ready
	if err := r.waitForUncors(); err != nil {
		return fmt.Errorf("uncors failed to start: %w", err)
	}

	return nil
}

// Teardown cleans up the test environment.
func (r *TestRunner) Teardown() {
	// Stop uncors
	if r.uncorsCmd != nil && r.uncorsCmd.Process != nil {
		_ = r.uncorsCmd.Process.Kill()
		_ = r.uncorsCmd.Wait()
	}

	// Clean up config file
	if r.uncorsConfigPath != "" {
		_ = os.Remove(r.uncorsConfigPath)
	}

	// Stop backend server
	if r.backendServer != nil {
		r.backendServer.Close()
	}
}

//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) generateUncorsConfig() error {
	config := r.buildConfigMap()

	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "uncors-test-*.yaml")
	if err != nil {
		return err
	}
	r.uncorsConfigPath = tmpFile.Name()

	// Write config as YAML
	encoder := yaml.NewEncoder(tmpFile)
	encoder.SetIndent(yamlIndent)
	if err := encoder.Encode(config); err != nil {
		tmpFile.Close()

		return err
	}
	tmpFile.Close()

	r.t.Logf("Generated uncors config at: %s", r.uncorsConfigPath)

	return nil
}

// buildConfigMap constructs the uncors configuration map.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) buildConfigMap() map[string]any {
	config := make(map[string]any)

	// Set ports if specified
	if r.testCase.Uncors.HTTPPort != 0 {
		config["http-port"] = r.testCase.Uncors.HTTPPort
	}
	if r.testCase.Uncors.HTTPSPort != 0 {
		config["https-port"] = r.testCase.Uncors.HTTPSPort
	}

	// Process mappings
	if len(r.testCase.Uncors.Mappings) > 0 {
		config["mappings"] = r.buildMappingsConfig()
	}

	// Merge with custom config
	for k, v := range r.testCase.Uncors.Config {
		config[k] = v
	}

	return config
}

// buildMappingsConfig constructs the mappings configuration array.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) buildMappingsConfig() []map[string]any {
	mappings := make([]map[string]any, 0, len(r.testCase.Uncors.Mappings))
	for _, mapping := range r.testCase.Uncors.Mappings {
		m := r.buildSingleMapping(mapping)
		mappings = append(mappings, m)
	}

	return mappings
}

// buildSingleMapping constructs a single mapping configuration.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) buildSingleMapping(mapping MappingConfig) map[string]any {
	mappingConfig := make(map[string]any)

	// Replace backend URL placeholder with actual server URL
	to := mapping.To
	if r.backendServer != nil && to == "{{backend}}" {
		to = r.backendServer.URL()
	}

	mappingConfig["from"] = mapping.From
	mappingConfig["to"] = to

	// Add optional configurations if present
	r.addIfNotEmpty(mappingConfig, "mocks", mapping.Mocks)
	r.addIfNotEmpty(mappingConfig, "statics", mapping.Statics)
	r.addIfNotEmpty(mappingConfig, "cache", mapping.Cache)
	r.addIfNotEmpty(mappingConfig, "rewrites", mapping.Rewrites)

	return mappingConfig
}

// addIfNotEmpty adds a value to the map if the slice is not empty.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) addIfNotEmpty(mappingConfig map[string]any, key string, value any) {
	// Use type switching to check if value is a slice and not empty
	switch val := value.(type) {
	case []MockConfig:
		if len(val) > 0 {
			mappingConfig[key] = val
		}
	case []StaticConfig:
		if len(val) > 0 {
			mappingConfig[key] = val
		}
	case []string:
		if len(val) > 0 {
			mappingConfig[key] = val
		}
	case []RewriteConfig:
		if len(val) > 0 {
			mappingConfig[key] = val
		}
	}
}

// startUncors starts the uncors proxy server.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) startUncors() error {
	// Find uncors binary
	uncorsBinary, err := r.findUncorsBinary()
	if err != nil {
		return err
	}

	// Start uncors with config
	r.uncorsCmd = exec.Command(uncorsBinary, "--config", r.uncorsConfigPath)
	r.uncorsCmd.Stdout = os.Stdout
	r.uncorsCmd.Stderr = os.Stderr

	if err := r.uncorsCmd.Start(); err != nil {
		return fmt.Errorf("failed to start uncors: %w", err)
	}

	r.t.Logf("Started uncors with PID: %d", r.uncorsCmd.Process.Pid)

	return nil
}

// findUncorsBinary locates the uncors binary.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) findUncorsBinary() (string, error) {
	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	binaryPath := filepath.Join(projectRoot, "uncors")

	// Check if binary already exists and is recent
	if info, err := os.Stat(binaryPath); err == nil {
		// Binary exists, check if it's newer than the source files
		if time.Since(info.ModTime()) < 5*time.Minute {
			r.t.Logf("Using existing uncors binary: %s", binaryPath)

			return binaryPath, nil
		}
	}

	// Build the binary from project root
	r.t.Logf("Building uncors binary...")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = projectRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build uncors: %w", err)
	}

	r.t.Logf("Built uncors binary at: %s", binaryPath)

	return binaryPath, nil
}

// waitForUncors waits for uncors to be ready to accept connections.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) waitForUncors() error {
	port := r.getHTTPPort()
	timeout := time.After(uncorsStartupTimeout)
	ticker := time.NewTicker(startupCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return ErrUncorsStartTimeout
		case <-ticker.C:
			// Try to connect
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				fmt.Sprintf("http://localhost:%d", port),
				http.NoBody,
			)
			if err != nil {
				continue
			}

			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				r.t.Logf("Uncors is ready on port %d", port)

				return nil
			}
		}
	}
}

// getHTTPPort returns the HTTP port for uncors, using default if not specified.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) getHTTPPort() int {
	if r.testCase.Uncors.HTTPPort != 0 {
		return r.testCase.Uncors.HTTPPort
	}

	return defaultHTTPPort
}

// collectRequests collects all requests to execute from the test definition.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) collectRequests(test TestDefinition) []RequestConfig {
	return test.Requests
}

// buildRequestURLFromConfig constructs the request URL from request config.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) buildRequestURLFromConfig(req RequestConfig) string {
	// Use explicit URL if provided
	if req.URL != "" {
		return req.URL
	}

	// Build URL from path
	if req.Path != "" {
		port := r.getHTTPPort()

		return fmt.Sprintf("http://localhost:%d%s", port, req.Path)
	}

	return ""
}

// Run executes all tests in the test case.
func (r *TestRunner) Run() {
	for _, test := range r.testCase.Tests {
		r.t.Run(test.Name, func(t *testing.T) {
			r.runTest(t, test)
		})
	}
}

// runTest executes a single test.
func (r *TestRunner) runTest(t *testing.T, test TestDefinition) {
	// Clear backend request history before test
	if r.backendServer != nil {
		r.backendServer.ClearRequests()
	}

	// Execute request sequence and get last response
	resp, bodyBytes := r.executeRequestSequence(t, test)
	defer resp.Body.Close()

	// Verify response from last request
	r.verifyResponse(t, test, resp, bodyBytes)

	// Verify backend call counts
	r.verifyBackendCalls(t, test)
}

// executeRequestSequence executes all requests for a test.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) executeRequestSequence(
	t *testing.T,
	test TestDefinition,
) (*http.Response, []byte) {
	requests := r.collectRequests(test)

	var resp *http.Response
	var bodyBytes []byte
	for reqIndex, reqConfig := range requests {
		req := r.createRequestFromConfig(t, reqConfig)
		resp, bodyBytes = r.executeRequest(t, req)
		// Close response for all but the last request
		if reqIndex < len(requests)-1 {
			resp.Body.Close()
		}
	}

	return resp, bodyBytes
}

// verifyBackendCalls verifies backend call counts match expectations.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) verifyBackendCalls(t *testing.T, test TestDefinition) {
	if r.backendServer == nil {
		return
	}

	// Verify endpoint-specific call counts
	if len(test.EndpointCallsCount) > 0 {
		r.verifyEndpointCallCounts(t, test)
	}
}

// verifyEndpointCallCounts verifies per-endpoint call counts.
//
//nolint:funcorder // Helper methods grouped logically with usage
func (r *TestRunner) verifyEndpointCallCounts(t *testing.T, test TestDefinition) {
	allRequests := r.backendServer.GetRequests()
	endpointCounts := make(map[string]int)
	for _, req := range allRequests {
		endpointCounts[req.Path]++
	}
	for endpoint, expectedCount := range test.EndpointCallsCount {
		actualCount := endpointCounts[endpoint]
		assert.Equal(t, expectedCount, actualCount,
			"expected %d call(s) to %s but got %d", expectedCount, endpoint, actualCount)
	}
}

// createRequestFromConfig builds an HTTP request from request configuration.
func (r *TestRunner) createRequestFromConfig(t *testing.T, reqConfig RequestConfig) *http.Request {
	url := r.buildRequestURLFromConfig(reqConfig)

	var bodyReader io.Reader
	if reqConfig.Body != "" {
		bodyReader = strings.NewReader(reqConfig.Body)
	}

	req, err := http.NewRequestWithContext(t.Context(), reqConfig.Method, url, bodyReader)
	require.NoError(t, err, "failed to create request")

	// Set headers
	for key, value := range reqConfig.Headers {
		req.Header.Set(key, value)
	}

	return req
}

// executeRequest executes the HTTP request and returns response with body.
func (r *TestRunner) executeRequest(t *testing.T, req *http.Request) (*http.Response, []byte) {
	ctx, cancel := context.WithTimeout(t.Context(), requestTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := r.httpClient.Do(req)
	require.NoError(t, err, "request failed")

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	return resp, bodyBytes
}

// verifyResponse checks all response expectations.
func (r *TestRunner) verifyResponse(t *testing.T, test TestDefinition, resp *http.Response, bodyBytes []byte) {
	// Verify status code
	if test.Expected.Status != 0 {
		assert.Equal(t, test.Expected.Status, resp.StatusCode, "status code mismatch")
	}

	// Verify body exact match
	if test.Expected.Body != "" {
		assert.Equal(t, test.Expected.Body, string(bodyBytes), "body mismatch")
	}

	// Verify body contains
	for _, substring := range test.Expected.BodyContains {
		assert.Contains(t, string(bodyBytes), substring, "body should contain substring")
	}

	// Verify body JSON
	r.verifyBodyJSON(t, test.Expected.BodyJSON, bodyBytes)

	// Verify headers
	for key, expectedValue := range test.Expected.Headers {
		assert.Equal(t, expectedValue, resp.Header.Get(key), "header %s mismatch", key)
	}

	// Verify headers exist
	for _, header := range test.Expected.HeadersExist {
		assert.NotEmpty(t, resp.Header.Get(header), "header %s should exist", header)
	}
}

// verifyBodyJSON verifies JSON response body against expected values.
func (r *TestRunner) verifyBodyJSON(t *testing.T, expected map[string]interface{}, bodyBytes []byte) {
	if len(expected) == 0 {
		return
	}

	var actualJSON map[string]interface{}
	err := json.Unmarshal(bodyBytes, &actualJSON)
	require.NoError(t, err, "failed to parse response as JSON")

	for key, expectedValue := range expected {
		assert.Equal(t, expectedValue, actualJSON[key], "JSON field %s mismatch", key)
	}
}

// findProjectRoot finds the project root directory (where the main module go.mod is).
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// Check if this is the main module (github.com/evg4b/uncors)
			if strings.Contains(string(data), "module github.com/evg4b/uncors") &&
				!strings.Contains(string(data), "module github.com/evg4b/uncors/") {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrProjectRootNotFound
		}
		dir = parent
	}
}
