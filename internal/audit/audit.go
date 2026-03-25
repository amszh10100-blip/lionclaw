package audit

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Logger struct {
	db *sql.DB
}

type Entry struct {
	ID        int64
	Timestamp time.Time
	UserID    string
	Action    string // "chat", "command", "skill_run", "config_change"
	Detail    string
	Model     string
	TokensIn  int
	TokensOut int
	Cost      float64
}

func NewLogger(dataDir string) (*Logger, error) {
	dbPath := filepath.Join(dataDir, "audit.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		user_id TEXT,
		action TEXT,
		detail TEXT,
		model TEXT,
		tokens_in INTEGER,
		tokens_out INTEGER,
		cost REAL
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &Logger{db: db}, nil
}

func (l *Logger) Log(entry Entry) error {
	query := `
	INSERT INTO audit_log (timestamp, user_id, action, detail, model, tokens_in, tokens_out, cost)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := l.db.Exec(query, entry.Timestamp, entry.UserID, entry.Action, entry.Detail, entry.Model, entry.TokensIn, entry.TokensOut, entry.Cost)
	return err
}

func (l *Logger) Query(since time.Time, limit int) ([]Entry, error) {
	query := `
	SELECT id, timestamp, user_id, action, detail, model, tokens_in, tokens_out, cost
	FROM audit_log
	WHERE timestamp > ?
	ORDER BY timestamp DESC
	LIMIT ?
	`
	rows, err := l.db.Query(query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		err := rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.Action, &e.Detail, &e.Model, &e.TokensIn, &e.TokensOut, &e.Cost)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (l *Logger) Export(w io.Writer) error {
	query := `
	SELECT id, timestamp, user_id, action, detail, model, tokens_in, tokens_out, cost
	FROM audit_log
	ORDER BY timestamp ASC
	`
	rows, err := l.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	writer := csv.NewWriter(w)
	header := []string{"ID", "Timestamp", "UserID", "Action", "Detail", "Model", "TokensIn", "TokensOut", "Cost"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for rows.Next() {
		var e Entry
		err := rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.Action, &e.Detail, &e.Model, &e.TokensIn, &e.TokensOut, &e.Cost)
		if err != nil {
			return err
		}

		record := []string{
			strconv.FormatInt(e.ID, 10),
			e.Timestamp.Format(time.RFC3339),
			e.UserID,
			e.Action,
			e.Detail,
			e.Model,
			strconv.Itoa(e.TokensIn),
			strconv.Itoa(e.TokensOut),
			fmt.Sprintf("%f", e.Cost),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func (l *Logger) Close() error {
	return l.db.Close()
}
