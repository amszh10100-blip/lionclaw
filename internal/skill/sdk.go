package skill

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Create 创建新 Skill 脚手架
func Create(baseDir, name string) error {
	dir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// 生成 skill.yaml
	manifest := Manifest{
		Name:        name,
		Version:     "0.1.0",
		Description: fmt.Sprintf("%s — GoldLion Skill", name),
		Permissions: Permissions{
			Network:    []string{},
			Filesystem: "none",
		},
		Entrypoint: "run.sh",
		Triggers: []Trigger{
			{Type: "keyword", Value: name},
		},
	}

	data, _ := yaml.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "skill.yaml"), data, 0600); err != nil {
		return err
	}

	// 生成入口脚本
	script := `#!/bin/bash
# GoldLion Skill: ` + name + `
# 从 stdin 读取输入，输出到 stdout

INPUT=$(cat)
echo "Skill '` + name + `' 收到输入: $INPUT"
echo "TODO: 实现你的逻辑"
`
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte(script), 0755); err != nil {
		return err
	}

	// 生成 README
	readme := fmt.Sprintf("# %s\n\nGoldLion Skill\n\n## 使用\n\n在对话中提到 `%s` 即可触发。\n\n## 权限\n\n- 网络: 无\n- 文件: 无\n", name, name)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0600); err != nil {
		return err
	}

	// 生成测试
	test := `#!/bin/bash
# 测试 Skill
echo "测试输入" | ./run.sh
`
	if err := os.WriteFile(filepath.Join(dir, "test.sh"), []byte(test), 0755); err != nil {
		return err
	}

	return nil
}

// Audit 安全审计 Skill
func Audit(dir string) ([]AuditResult, error) {
	var results []AuditResult

	// 检查 manifest
	mPath := filepath.Join(dir, "skill.yaml")
	data, err := os.ReadFile(mPath)
	if err != nil {
		return nil, fmt.Errorf("skill.yaml 不存在")
	}

	var m Manifest
	if err := parseManifest(data, &m); err != nil {
		results = append(results, AuditResult{"manifest", "error", "YAML 解析失败"})
		return results, nil
	}

	results = append(results, AuditResult{"manifest", "pass", "格式正确"})

	// 检查权限声明
	if m.Permissions.Filesystem == "write" {
		results = append(results, AuditResult{"filesystem", "warn", "需要写权限"})
	} else {
		results = append(results, AuditResult{"filesystem", "pass", fmt.Sprintf("权限: %s", m.Permissions.Filesystem)})
	}

	if len(m.Permissions.Network) > 0 {
		results = append(results, AuditResult{"network", "warn", fmt.Sprintf("需要网络: %v", m.Permissions.Network)})
	} else {
		results = append(results, AuditResult{"network", "pass", "无网络需求"})
	}

	if len(m.Permissions.Credentials) > 0 {
		results = append(results, AuditResult{"credentials", "warn", fmt.Sprintf("需要凭证: %v", m.Permissions.Credentials)})
	} else {
		results = append(results, AuditResult{"credentials", "pass", "无凭证需求"})
	}

	// 扫描危险模式
	entrypoint := filepath.Join(dir, m.Entrypoint)
	if code, err := os.ReadFile(entrypoint); err == nil {
		dangerPatterns := []string{
			"curl ", "wget ", "nc ", "ncat ",
			"/etc/passwd", "/etc/shadow",
			"eval(", "exec(",
			"rm -rf", "dd if=",
		}
		for _, p := range dangerPatterns {
			if containsStr(string(code), p) {
				results = append(results, AuditResult{"code_scan", "warn", fmt.Sprintf("检测到危险模式: %s", p)})
			}
		}
		if len(results) == 4 { // 只有前 4 项，没有新增警告
			results = append(results, AuditResult{"code_scan", "pass", "无危险模式"})
		}
	}

	return results, nil
}

// AuditResult 审计结果
type AuditResult struct {
	Check  string `json:"check"`
	Status string `json:"status"` // "pass", "warn", "error"
	Detail string `json:"detail"`
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findStr(s, substr))
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
