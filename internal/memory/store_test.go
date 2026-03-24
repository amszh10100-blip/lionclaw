package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteStore_SaveAndGet(t *testing.T) {
	dir := t.TempDir()

	store, err := NewSQLiteStore(dir)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	// Save
	store.SaveMessage("sess1", Entry{Role: "user", Content: "你好"})
	store.SaveMessage("sess1", Entry{Role: "assistant", Content: "你好！有什么可以帮你的？"})
	store.SaveMessage("sess1", Entry{Role: "user", Content: "帮我分析项目预算"})

	// Get
	history, err := store.GetHistory("sess1", 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("history = %d, want 3", len(history))
	}

	// 顺序：最旧在前
	if history[0].Content != "你好" {
		t.Errorf("first message = %q, want '你好'", history[0].Content)
	}
	if history[2].Content != "帮我分析项目预算" {
		t.Errorf("last message = %q", history[2].Content)
	}
}

func TestSQLiteStore_Search(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSQLiteStore(dir)

	store.SaveMessage("s1", Entry{Role: "user", Content: "LionClaw 的安全架构很好"})
	store.SaveMessage("s1", Entry{Role: "user", Content: "今天天气不错"})
	store.SaveMessage("s1", Entry{Role: "user", Content: "LionClaw 的成本控制也很强"})

	results, err := store.Search("LionClaw", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("search results = %d, want 2", len(results))
	}
}

func TestSQLiteStore_SessionIsolation(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSQLiteStore(dir)

	store.SaveMessage("user-A", Entry{Role: "user", Content: "A的消息"})
	store.SaveMessage("user-B", Entry{Role: "user", Content: "B的消息"})

	histA, _ := store.GetHistory("user-A", 10)
	histB, _ := store.GetHistory("user-B", 10)

	if len(histA) != 1 || histA[0].Content != "A的消息" {
		t.Errorf("user-A history wrong: %v", histA)
	}
	if len(histB) != 1 || histB[0].Content != "B的消息" {
		t.Errorf("user-B history wrong: %v", histB)
	}
}

func TestSQLiteStore_ExportMarkdown(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSQLiteStore(dir)

	store.SaveMessage("s1", Entry{Role: "user", Content: "测试导出"})
	store.SaveMessage("s1", Entry{Role: "assistant", Content: "导出成功"})

	exportPath := filepath.Join(dir, "export.md")
	if err := store.ExportMarkdown(exportPath); err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}

	data, _ := os.ReadFile(exportPath)
	content := string(data)

	if len(content) == 0 {
		t.Error("exported file is empty")
	}
	if !containsString(content, "测试导出") {
		t.Error("export missing '测试导出'")
	}
	if !containsString(content, "LionClaw Memory Export") {
		t.Error("export missing header")
	}
}

func TestSQLiteStore_Limit(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewSQLiteStore(dir)

	for i := 0; i < 50; i++ {
		store.SaveMessage("s1", Entry{Role: "user", Content: "msg"})
	}

	hist, _ := store.GetHistory("s1", 10)
	if len(hist) != 10 {
		t.Errorf("limit 10, got %d", len(hist))
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && findString(s, sub)
}

func findString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
