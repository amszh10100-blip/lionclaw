package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Entry 记忆条目
type Entry struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Summary   string    `json:"summary,omitempty"`
	Tokens    int       `json:"tokens"`
	Model     string    `json:"model,omitempty"`
	CostUSD   float64   `json:"cost_usd,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Store 记忆存储接口
type Store interface {
	SaveMessage(sessionID string, entry Entry) error
	GetHistory(sessionID string, limit int) ([]Entry, error)
	Search(query string, limit int) ([]Entry, error)
	ExportMarkdown(path string) error
	ImportMarkdown(path string) error
}

// SQLiteStore SQLite 实现
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dataDir string) (*SQLiteStore, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "goldlion.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			role        TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
			content     TEXT NOT NULL,
			summary     TEXT,
			tokens      INTEGER DEFAULT 0,
			model       TEXT,
			cost_usd    REAL DEFAULT 0,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, created_at);
	`); err != nil {
		return nil, fmt.Errorf("创建表失败: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) SaveMessage(sessionID string, entry Entry) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (session_id, role, content, summary, tokens, model, cost_usd, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, entry.Role, entry.Content, entry.Summary,
		entry.Tokens, entry.Model, entry.CostUSD, time.Now(),
	)
	return err
}

func (s *SQLiteStore) GetHistory(sessionID string, limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, summary, tokens, model, cost_usd, created_at
		 FROM messages WHERE session_id = ? ORDER BY created_at DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Role, &e.Content, &e.Summary, &e.Tokens, &e.Model, &e.CostUSD, &ts); err != nil {
			continue
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
		entries = append(entries, e)
	}

	// 反转顺序（最旧的在前）
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

func (s *SQLiteStore) Search(query string, limit int) ([]Entry, error) {
	// P0: 简单 LIKE 搜索
	// P1: 升级为 FTS5 全文检索
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, summary, tokens, model, cost_usd, created_at
		 FROM messages WHERE content LIKE ? ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Role, &e.Content, &e.Summary, &e.Tokens, &e.Model, &e.CostUSD, &ts); err != nil {
			continue
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *SQLiteStore) ExportMarkdown(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	rows, err := s.db.Query(
		`SELECT role, content, created_at FROM messages ORDER BY created_at ASC LIMIT 10000`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# GoldLion Memory Export\n\n")
	fmt.Fprintf(f, "> Exported at %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	for rows.Next() {
		var role, content, ts string
		if err := rows.Scan(&role, &content, &ts); err != nil {
			continue
		}
		fmt.Fprintf(f, "## [%s] %s\n\n%s\n\n---\n\n", ts, role, content)
	}

	return nil
}

func (s *SQLiteStore) ImportMarkdown(path string) error {
	// TODO: P1 实现 Markdown 导入
	return fmt.Errorf("Markdown 导入尚未实现")
}
