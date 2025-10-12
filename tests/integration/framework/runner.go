package framework

import (
	"context"
	"encoding/json"
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

// TestRunner orchestrates integration tests.
type TestRunner struct {
	t              *testing.T
	testCase       *TestCase
	backendServer  *TestServer
	uncorsCmd      *exec.Cmd
	uncorsConfigPath string
	httpClient     *http.Client
	resolver       *DNSResolver
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

	// Create DNS resolver with default localhost mapping
	r.resolver = NewDNSResolver(map[string]string{
		"localhost": "127.0.0.1",
	})

	// Create HTTP client with custom DNS resolver
	r.httpClient = CreateHTTPClient(r.resolver)

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
		r.uncorsCmd.Process.Kill()
		r.uncorsCmd.Wait()
	}

	// Clean up config file
	if r.uncorsConfigPath != "" {
		os.Remove(r.uncorsConfigPath)
	}

	// Stop backend server
	if r.backendServer != nil {
		r.backendServer.Close()
	}
}

// generateUncorsConfig creates a configuration file for uncors.
func (r *TestRunner) generateUncorsConfig() error {
	config := make(map[string]any)

	// Set HTTP port
	if r.testCase.Uncors.HTTPPort != 0 {
		config["http-port"] = r.testCase.Uncors.HTTPPort
	}

	// Set HTTPS port
	if r.testCase.Uncors.HTTPSPort != 0 {
		config["https-port"] = r.testCase.Uncors.HTTPSPort
	}

	// Process mappings
	if len(r.testCase.Uncors.Mappings) > 0 {
		mappings := make([]map[string]any, 0, len(r.testCase.Uncors.Mappings))
		for _, mapping := range r.testCase.Uncors.Mappings {
			m := make(map[string]any)

			// Replace backend URL placeholder with actual server URL
			from := mapping.From
			to := mapping.To
			if r.backendServer != nil && to == "{{backend}}" {
				to = r.backendServer.URL()
			}

			m["from"] = from
			m["to"] = to

			if len(mapping.Mocks) > 0 {
				m["mocks"] = mapping.Mocks
			}
			if len(mapping.Statics) > 0 {
				m["statics"] = mapping.Statics
			}
			if len(mapping.Cache) > 0 {
				m["cache"] = mapping.Cache
			}
			if len(mapping.Rewrites) > 0 {
				m["rewrites"] = mapping.Rewrites
			}

			mappings = append(mappings, m)
		}
		config["mappings"] = mappings
	}

	// Merge with custom config
	for k, v := range r.testCase.Uncors.Config {
		config[k] = v
	}

	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "uncors-test-*.yaml")
	if err != nil {
		return err
	}
	r.uncorsConfigPath = tmpFile.Name()

	// Write config as YAML
	encoder := yaml.NewEncoder(tmpFile)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	r.t.Logf("Generated uncors config at: %s", r.uncorsConfigPath)
	return nil
}

// startUncors starts the uncors proxy server.
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
func (r *TestRunner) waitForUncors() error {
	port := r.testCase.Uncors.HTTPPort
	if port == 0 {
		port = 3000 // default port
	}

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for uncors to start")
		case <-ticker.C:
			// Try to connect
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
			if err == nil {
				resp.Body.Close()
				r.t.Logf("Uncors is ready on port %d", port)
				return nil
			}
		}
	}
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
	// Update DNS resolver with test-specific mappings
	for host, ip := range test.DNS {
		r.resolver.AddMapping(host, ip)
	}

	// Build request URL
	url := test.Request.URL
	if url == "" && test.Request.Path != "" {
		port := r.testCase.Uncors.HTTPPort
		if port == 0 {
			port = 3000
		}
		url = fmt.Sprintf("http://localhost:%d%s", port, test.Request.Path)
	}

	// Create request
	var bodyReader io.Reader
	if test.Request.Body != "" {
		bodyReader = strings.NewReader(test.Request.Body)
	}

	req, err := http.NewRequest(test.Request.Method, url, bodyReader)
	require.NoError(t, err, "failed to create request")

	// Set headers
	for key, value := range test.Request.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := r.httpClient.Do(req)
	require.NoError(t, err, "request failed")
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

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
	if len(test.Expected.BodyJSON) > 0 {
		var actualJSON map[string]interface{}
		err := json.Unmarshal(bodyBytes, &actualJSON)
		require.NoError(t, err, "failed to parse response as JSON")

		for key, expectedValue := range test.Expected.BodyJSON {
			assert.Equal(t, expectedValue, actualJSON[key], "JSON field %s mismatch", key)
		}
	}

	// Verify headers
	for key, expectedValue := range test.Expected.Headers {
		assert.Equal(t, expectedValue, resp.Header.Get(key), "header %s mismatch", key)
	}

	// Verify headers exist
	for _, header := range test.Expected.HeadersExist {
		assert.NotEmpty(t, resp.Header.Get(header), "header %s should exist", header)
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
			return "", fmt.Errorf("project root not found")
		}
		dir = parent
	}
}
