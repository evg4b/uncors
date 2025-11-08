package server_test

import (
	"context"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/server"
)

func TestServer(t *testing.T) {
	s := server.New()

	mapping := config.Mappings{
		config.Mapping{From: "http://localhost:3000", To: "https://gihub.com"},
		config.Mapping{From: "http://localhost:4000", To: "https://gihub.com"},
		config.Mapping{From: "http://localhost:5000", To: "https://gihub.com"},
	}

	s.Start(context.Background(), mapping.GroupByPort())

	s.Waite()
}
