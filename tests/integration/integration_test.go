package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/tests/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// Load all test cases from testcases directory
	testCasesDir := filepath.Join("testcases")
	testCases, err := framework.LoadTestCasesFromDir(testCasesDir)
	require.NoError(t, err, "failed to load test cases")

	if len(testCases) == 0 {
		t.Skip("no test cases found")
	}

	// Run each test case
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			runner := framework.NewTestRunner(t, testCase)

			// Setup test environment
			err := runner.Setup()
			require.NoError(t, err, "failed to setup test environment")
			defer runner.Teardown()

			// Run all tests
			runner.Run()
		})
	}
}
