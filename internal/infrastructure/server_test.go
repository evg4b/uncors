package infrastructure_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	t.Run("provide correct parameters", func(t *testing.T) {
		server := infrastructure.NewServer("0.0.0.0", 3000, http.DefaultServeMux)

		assert.NotNil(t, server)
		assert.Equal(t, http.DefaultServeMux, server.Handler)
		assert.Equal(t, "0.0.0.0:3000", server.Addr)
	})
}
