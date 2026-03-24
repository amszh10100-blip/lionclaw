package brain

import (
	"log/slog"
	"testing"

	"github.com/lionclaw/lionclaw/internal/config"
)

func newTestRouter() *DefaultRouter {
	cfg := config.DefaultConfig()
	cost := &mockCostTracker{budget: Budget{DailyLimitUSD: 5, MonthlyLimitUSD: 50, WarnAtPercent: 0.8}}
	logger := slog.Default()
	r, _ := NewRouter(cfg, cost, logger)
	return r
}

// mockCostTracker 测试用成本追踪
type mockCostTracker struct {
	budget  Budget
	total   float64
	records []CostRecord
}

func (m *mockCostTracker) Record(r CostRecord) error {
	m.records = append(m.records, r)
	m.total += r.CostUSD
	return nil
}
func (m *mockCostTracker) GetToday() (float64, []CostRecord, error) { return m.total, m.records, nil }
func (m *mockCostTracker) GetMonth() (float64, []CostRecord, error) { return m.total, m.records, nil }
func (m *mockCostTracker) CheckBudget(est float64) (bool, float64, error) {
	remaining := m.budget.DailyLimitUSD - m.total
	return remaining > est, remaining, nil
}
func (m *mockCostTracker) GetBudget() Budget        { return m.budget }
func (m *mockCostTracker) SetBudget(b Budget) error { m.budget = b; return nil }

func TestAssessComplexity_Greeting(t *testing.T) {
	r := newTestRouter()
	tests := []struct {
		input    string
		expected Complexity
	}{
		{"你好", ComplexityLow},
		{"hi", ComplexityLow},
		{"hello", ComplexityLow},
		{"ok", ComplexityLow},
		{"谢谢", ComplexityLow},
	}

	for _, tt := range tests {
		got := r.assessComplexity(tt.input, nil)
		if got != tt.expected {
			t.Errorf("assessComplexity(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestAssessComplexity_High(t *testing.T) {
	r := newTestRouter()
	tests := []struct {
		input    string
		expected Complexity
	}{
		{"为什么这个架构不好？请帮我分析一下", ComplexityHigh},
		{"帮我设计一个微服务架构，比较 Go 和 Rust 的优缺点", ComplexityHigh},
		{"请写一段代码实现快速排序算法", ComplexityHigh},
	}

	for _, tt := range tests {
		got := r.assessComplexity(tt.input, nil)
		if got != tt.expected {
			t.Errorf("assessComplexity(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestAssessComplexity_Medium(t *testing.T) {
	r := newTestRouter()
	tests := []struct {
		input string
	}{
		{"怎么安装 Docker？"},
		{"请问今天天气如何"},
		{"帮我翻译这段话"},
	}

	for _, tt := range tests {
		got := r.assessComplexity(tt.input, nil)
		if got < ComplexityLow || got > ComplexityHigh {
			t.Errorf("assessComplexity(%q) = %d, out of range", tt.input, got)
		}
	}
}

func TestPrivacyKeywords(t *testing.T) {
	r := newTestRouter()

	tests := []struct {
		input    string
		expected bool
	}{
		{"我的密码是 123456", true},
		{"帮我查一下 password", true},
		{"我的银行卡号是xxx", true},
		{"token 是什么意思", true},
		{"今天天气怎么样", false},
		{"帮我写个函数", false},
	}

	for _, tt := range tests {
		got := r.containsPrivacyKeywords(tt.input)
		if got != tt.expected {
			t.Errorf("containsPrivacyKeywords(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsGreeting(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"你好", true},
		{"hi", true},
		{"Hello", true},
		{"ok", true},
		{"你好，帮我做件事", false},
		{"分析一下", false},
	}

	for _, tt := range tests {
		got := isGreeting(tt.input)
		if got != tt.expected {
			t.Errorf("isGreeting(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
