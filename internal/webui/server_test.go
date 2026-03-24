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
