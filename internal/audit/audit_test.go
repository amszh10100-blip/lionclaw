package audit

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	if logger.db == nil {
		t.Errorf("expected db to be initialized")
	}
}

func TestLogAndQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	now := time.Now()
	entry := Entry{
		Timestamp: now,
		UserID:    "user1",
		Action:    "chat",
		Detail:    "test detail",
		Model:     "model1",
		TokensIn:  10,
		TokensOut: 20,
		Cost:      0.01,
	}

	err = logger.Log(entry)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	since := now.Add(-time.Hour)
	entries, err := logger.Query(since, 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.UserID != entry.UserID || e.Action != entry.Action || e.Detail != entry.Detail || e.Model != entry.Model || e.TokensIn != entry.TokensIn || e.TokensOut != entry.TokensOut || e.Cost != entry.Cost {
		t.Errorf("entry mismatch: %+v", e)
	}
}

func TestExport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	now := time.Now()
	entry := Entry{
		Timestamp: now,
		UserID:    "user2",
		Action:    "command",
		Detail:    "/test",
		Model:     "model2",
		TokensIn:  5,
		TokensOut: 15,
		Cost:      0.02,
	}

	err = logger.Log(entry)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	var buf bytes.Buffer
	err = logger.Export(&buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	csvStr := buf.String()
	if !strings.Contains(csvStr, "ID,Timestamp,UserID,Action,Detail,Model,TokensIn,TokensOut,Cost") {
		t.Errorf("missing header in CSV")
	}
	if !strings.Contains(csvStr, "user2") || !strings.Contains(csvStr, "command") {
		t.Errorf("missing data in CSV: %s", csvStr)
	}
}

func TestQueryEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	entries, err := logger.Query(time.Now(), 10)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}