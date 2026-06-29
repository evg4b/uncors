package cli_test

import (
	"os"
	"testing"

	"github.com/evg4b/uncors/testing/integration"
)

func TestMain(m *testing.M) {
	integration.SetupBin(m)
	os.Exit(m.Run())
}
