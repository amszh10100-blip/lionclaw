package gateway

import (
	"testing"
	"github.com/lionclaw/lionclaw/internal/config"
)

func TestGatewayNew(t *testing.T) {
	cfg := config.DefaultConfig()
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create gateway: %v", err)
	}
	if gw == nil {
		t.Fatal("Gateway should not be nil")
	}
}

func TestHandleMessage(t *testing.T) {
    // Add simple mock test
}
