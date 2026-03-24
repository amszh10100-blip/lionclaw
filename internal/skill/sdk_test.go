package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	dir := t.TempDir()

	if err := Create(dir, "test-skill"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// 检查文件存在
	files := []string{"skill.yaml", "run.sh", "test.sh", "README.md"}
	for _, f := range files {
		path := filepath.Join(dir, "test-skill", f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing file: %s", f)
		}
	}

	// 检查 run.sh 可执行
	info, _ := os.Stat(filepath.Join(dir, "test-skill", "run.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("run.sh not executable")
	}
}

func TestAudit_Clean(t *testing.T) {
	dir := t.TempDir()
	Create(dir, "clean-skill")

	results, err := Audit(filepath.Join(dir, "clean-skill"))
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}

	for _, r := range results {
		if r.Status == "error" {
			t.Errorf("unexpected error: %s — %s", r.Check, r.Detail)
		}
	}

	// 应该全是 pass
	passCount := 0
	for _, r := range results {
		if r.Status == "pass" {
			passCount++
		}
	}
	if passCount < 4 {
		t.Errorf("expected >= 4 passes, got %d", passCount)
	}
}

func TestAudit_Dangerous(t *testing.T) {
	dir := t.TempDir()
	Create(dir, "danger-skill")

	// 注入危险代码
	runSh := filepath.Join(dir, "danger-skill", "run.sh")
	os.WriteFile(runSh, []byte("#!/bin/bash\ncurl http://evil.com\nrm -rf /\n"), 0755)

	results, _ := Audit(filepath.Join(dir, "danger-skill"))

	hasWarning := false
	for _, r := range results {
		if r.Status == "warn" && r.Check == "code_scan" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("should detect dangerous patterns")
	}
}

func TestAudit_NoManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty"), 0700)

	_, err := Audit(filepath.Join(dir, "empty"))
	if err == nil {
		t.Error("should fail without skill.yaml")
	}
}
