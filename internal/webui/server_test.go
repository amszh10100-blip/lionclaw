package webui

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/lionclaw/lionclaw/internal/config"
	"github.com/lionclaw/lionclaw/internal/brain"
	"github.com/lionclaw/lionclaw/internal/memory"
)

type MockCostTracker struct{}
func (m *MockCostTracker) Record(record brain.CostRecord) error { return nil }
func (m *MockCostTracker) GetToday() (float64, []brain.CostRecord, error) { return 0, nil, nil }
func (m *MockCostTracker) GetMonth() (float64, []brain.CostRecord, error) { return 0, nil, nil }
func (m *MockCostTracker) CheckBudget(estimated float64) (bool, float64, error) { return true, 0, nil }
func (m *MockCostTracker) GetBudget() brain.Budget { return brain.Budget{} }
func (m *MockCostTracker) SetBudget(b brain.Budget) error { return nil }

type MockMemoryStore struct{}
func (m *MockMemoryStore) SaveMessage(sessionID string, entry memory.Entry) error { return nil }
func (m *MockMemoryStore) GetHistory(sessionID string, limit int) ([]memory.Entry, error) { return nil, nil }
func (m *MockMemoryStore) GetRecent(limit int) ([]memory.Entry, error) { return nil, nil }
func (m *MockMemoryStore) Search(query string, limit int) ([]memory.Entry, error) { return nil, nil }
func (m *MockMemoryStore) ExportMarkdown(path string) error { return nil }
func (m *MockMemoryStore) ImportMarkdown(path string) error { return nil }

func TestDashboard(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()
	cost := &MockCostTracker{}
	mem := &MockMemoryStore{}
	s := New(cfg, cost, mem, func() string { return "assistant" }, logger)

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

func TestAPIEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()
	cost := &MockCostTracker{}
	mem := &MockMemoryStore{}
	s := New(cfg, cost, mem, func() string { return "assistant" }, logger)

	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/status", s.handleAPIStatus},
		{"/api/cost", s.handleAPICost},
		{"/api/history", s.handleAPIHistory},
	}

	for _, tc := range endpoints {
		req, err := http.NewRequest("GET", tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		
		rr := httptest.NewRecorder()
		tc.handler(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler for %s returned wrong status code: got %v want %v", tc.path, status, http.StatusOK)
		}
		if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("handler for %s returned wrong content type: got %v want application/json", tc.path, contentType)
		}
	}
}
