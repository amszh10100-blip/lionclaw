package brain

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goldlion/goldlion/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteCostTracker 基于 SQLite 的成本追踪
type SQLiteCostTracker struct {
	db     *sql.DB
	budget Budget
}

func NewSQLiteCostTracker(dataDir string, cfg config.CostConfig) (*SQLiteCostTracker, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "goldlion.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建表
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cost_records (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			model         TEXT NOT NULL,
			is_local      BOOLEAN DEFAULT 0,
			input_tokens  INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			cost_usd      REAL DEFAULT 0,
			task_label    TEXT,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_cost_date ON cost_records(created_at);
	`); err != nil {
		return nil, fmt.Errorf("创建表失败: %w", err)
	}

	return &SQLiteCostTracker{
		db: db,
		budget: Budget{
			DailyLimitUSD:   cfg.DailyLimitUSD,
			MonthlyLimitUSD: cfg.MonthlyLimitUSD,
			WarnAtPercent:   cfg.WarnAtPercent,
		},
	}, nil
}

func (t *SQLiteCostTracker) Record(record CostRecord) error {
	_, err := t.db.Exec(
		`INSERT INTO cost_records (model, is_local, input_tokens, output_tokens, cost_usd, task_label, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.Model, record.IsLocal, record.InputTokens, record.OutputTokens,
		record.CostUSD, record.TaskLabel, time.Now(),
	)
	return err
}

func (t *SQLiteCostTracker) GetToday() (float64, []CostRecord, error) {
	today := time.Now().Format("2006-01-02")
	return t.getRecordsSince(today + " 00:00:00")
}

func (t *SQLiteCostTracker) GetMonth() (float64, []CostRecord, error) {
	month := time.Now().Format("2006-01") + "-01 00:00:00"
	return t.getRecordsSince(month)
}

func (t *SQLiteCostTracker) getRecordsSince(since string) (float64, []CostRecord, error) {
	rows, err := t.db.Query(
		`SELECT model, is_local, input_tokens, output_tokens, cost_usd, task_label, created_at
		 FROM cost_records WHERE created_at >= ? ORDER BY created_at DESC`,
		since,
	)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var total float64
	var records []CostRecord
	for rows.Next() {
		var r CostRecord
		var ts string
		if err := rows.Scan(&r.Model, &r.IsLocal, &r.InputTokens, &r.OutputTokens, &r.CostUSD, &r.TaskLabel, &ts); err != nil {
			continue
		}
		r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		total += r.CostUSD
		records = append(records, r)
	}
	return total, records, nil
}

func (t *SQLiteCostTracker) CheckBudget(estimated float64) (bool, float64, error) {
	todayTotal, _, err := t.GetToday()
	if err != nil {
		return true, 0, err // 出错时默认允许
	}

	remaining := t.budget.DailyLimitUSD - todayTotal
	if remaining <= 0 {
		return false, remaining, nil
	}
	if todayTotal+estimated > t.budget.DailyLimitUSD {
		return false, remaining, nil
	}
	return true, remaining, nil
}

func (t *SQLiteCostTracker) GetBudget() Budget {
	return t.budget
}

func (t *SQLiteCostTracker) SetBudget(b Budget) error {
	t.budget = b
	return nil
}
