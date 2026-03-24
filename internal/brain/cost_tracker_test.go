package brain

import (
	"testing"

	"github.com/goldlion/goldlion/internal/config"
)

func TestCostTracker_RecordAndGet(t *testing.T) {
	dir := t.TempDir()
	cfg := config.CostConfig{DailyLimitUSD: 5, MonthlyLimitUSD: 50, WarnAtPercent: 0.8}

	tracker, err := NewSQLiteCostTracker(dir, cfg)
	if err != nil {
		t.Fatalf("NewSQLiteCostTracker: %v", err)
	}

	// Record
	tracker.Record(CostRecord{Model: "qwen3:8b", IsLocal: true, InputTokens: 100, OutputTokens: 50, CostUSD: 0})
	tracker.Record(CostRecord{Model: "claude-opus", IsLocal: false, InputTokens: 500, OutputTokens: 200, CostUSD: 0.05})

	// GetToday
	total, records, err := tracker.GetToday()
	if err != nil {
		t.Fatalf("GetToday: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("records = %d, want 2", len(records))
	}
	if total != 0.05 {
		t.Errorf("total = %f, want 0.05", total)
	}
}

func TestCostTracker_Budget(t *testing.T) {
	dir := t.TempDir()
	cfg := config.CostConfig{DailyLimitUSD: 1.0, MonthlyLimitUSD: 10, WarnAtPercent: 0.8}

	tracker, _ := NewSQLiteCostTracker(dir, cfg)

	// 在预算内
	allowed, remaining, _ := tracker.CheckBudget(0.5)
	if !allowed {
		t.Error("should be allowed")
	}
	if remaining != 1.0 {
		t.Errorf("remaining = %f, want 1.0", remaining)
	}

	// 花掉 0.8
	tracker.Record(CostRecord{CostUSD: 0.8})

	// 还能花 0.1
	allowed, _, _ = tracker.CheckBudget(0.1)
	if !allowed {
		t.Error("0.1 should still be allowed")
	}

	// 但不能花 0.3（会超限）
	allowed, _, _ = tracker.CheckBudget(0.3)
	if allowed {
		t.Error("0.3 should not be allowed (would exceed 1.0)")
	}
}

func TestCostTracker_BudgetGetSet(t *testing.T) {
	dir := t.TempDir()
	cfg := config.CostConfig{DailyLimitUSD: 5, MonthlyLimitUSD: 50, WarnAtPercent: 0.8}

	tracker, _ := NewSQLiteCostTracker(dir, cfg)

	b := tracker.GetBudget()
	if b.DailyLimitUSD != 5 {
		t.Errorf("DailyLimit = %f, want 5", b.DailyLimitUSD)
	}

	tracker.SetBudget(Budget{DailyLimitUSD: 10, MonthlyLimitUSD: 100, WarnAtPercent: 0.9})
	b2 := tracker.GetBudget()
	if b2.DailyLimitUSD != 10 {
		t.Errorf("after set, DailyLimit = %f, want 10", b2.DailyLimitUSD)
	}
}
