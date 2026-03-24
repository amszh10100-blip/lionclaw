package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertOpenClawSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0700)

	// 写一个模拟的 OpenClaw SKILL.md
	skillMD := `---
name: weather
description: Get weather forecasts
version: 1.2.0
---
# Weather Skill

Use web_search to find weather data.
Then write the result to a file.
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0600)

	m, err := ConvertOpenClawSkill(skillDir)
	if err != nil {
		t.Fatalf("ConvertOpenClawSkill: %v", err)
	}

	if m.Name != "weather" {
		t.Errorf("name = %s, want weather", m.Name)
	}
	if m.Description != "Get weather forecasts" {
		t.Errorf("desc = %s", m.Description)
	}
	// 应检测到 web_search → 需要网络
	if len(m.Permissions.Network) == 0 {
		t.Error("should detect network permission from web_search")
	}
	// 应检测到 write → 需要文件写
	if m.Permissions.Filesystem != "write" {
		t.Errorf("filesystem = %s, want write", m.Permissions.Filesystem)
	}
}

func TestConvertOpenClawSkill_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "simple")
	os.MkdirAll(skillDir, 0700)

	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Simple Skill\nJust do something.\n"), 0600)

	m, err := ConvertOpenClawSkill(skillDir)
	if err != nil {
		t.Fatalf("ConvertOpenClawSkill: %v", err)
	}

	if m.Name != "simple" {
		t.Errorf("name = %s, want simple (dir name)", m.Name)
	}
}

func TestBatchConvert(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// 创建 3 个模拟 skill
	for _, name := range []string{"skill-a", "skill-b", "skill-c"} {
		d := filepath.Join(srcDir, name)
		os.MkdirAll(d, 0700)
		os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("# "+name), 0600)
	}
	// 一个没有 SKILL.md 的目录
	os.MkdirAll(filepath.Join(srcDir, "not-a-skill"), 0700)

	converted, skipped, errs := BatchConvert(srcDir, dstDir)

	if converted != 3 {
		t.Errorf("converted = %d, want 3", converted)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
	if len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}

	// 检查输出
	for _, name := range []string{"skill-a", "skill-b", "skill-c"} {
		yamlPath := filepath.Join(dstDir, name, "skill.yaml")
		if _, err := os.Stat(yamlPath); err != nil {
			t.Errorf("missing %s/skill.yaml", name)
		}
	}
}
