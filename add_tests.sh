#!/bin/bash
set -e
cd /Users/app/.openclaw/workspace/projects/lionclaw/src

mkdir -p internal/gateway
cat << 'TEST_EOF' > internal/gateway/gateway_test.go
package gateway

import (
	"context"
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
TEST_EOF

mkdir -p internal/webui
cat << 'TEST_EOF' > internal/webui/server_test.go
package webui

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/lionclaw/lionclaw/internal/config"
	"github.com/lionclaw/lionclaw/internal/brain"
)

type MockCostTracker struct{}
func (m *MockCostTracker) Record(record brain.CostRecord) error { return nil }
func (m *MockCostTracker) GetToday() (float64, []brain.CostRecord, error) { return 0, nil, nil }
func (m *MockCostTracker) GetMonth() (float64, []brain.CostRecord, error) { return 0, nil, nil }
func (m *MockCostTracker) CheckBudget(estimated float64) (bool, float64, error) { return true, 0, nil }
func (m *MockCostTracker) GetBudget() brain.Budget { return brain.Budget{} }
func (m *MockCostTracker) SetBudget(b brain.Budget) error { return nil }

func TestDashboard(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()
	cost := &MockCostTracker{}
	s := New(cfg, cost, logger)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
    // Set Basic Auth
    req.SetBasicAuth("admin", "lionclaw")

	rr := httptest.NewRecorder()
    handler := s.basicAuth(s.handleDashboard)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
TEST_EOF

mkdir -p internal/channel/telegram
cat << 'TEST_EOF' > internal/channel/telegram/bot_test.go
package telegram

import (
	"testing"
	"log/slog"
)

func TestNewBot(t *testing.T) {
	logger := slog.Default()
	bot := New("dummy_token", logger)
	if bot == nil {
		t.Fatal("Bot should not be nil")
	}
}
TEST_EOF

mkdir -p internal/migrate/openclaw
cat << 'TEST_EOF' > internal/migrate/openclaw/migration_test.go
package openclaw

import (
	"testing"
)

func TestMigration(t *testing.T) {
	// A placeholder for migration logic test
}
TEST_EOF

