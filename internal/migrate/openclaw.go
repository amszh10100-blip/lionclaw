package migrate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/goldlion/goldlion/internal/config"
	"github.com/goldlion/goldlion/internal/memory"
	"github.com/goldlion/goldlion/internal/vault"
)

// Result 迁移结果
type Result struct {
	MemoryEntries  int      `json:"memory_entries"`
	SkillsMigrated int      `json:"skills_migrated"`
	SkillsSkipped  int      `json:"skills_skipped"`
	ConfigMigrated bool     `json:"config_migrated"`
	SecurityFixes  []string `json:"security_fixes"`
	Warnings       []string `json:"warnings"`
}

// OpenClaw 从 OpenClaw 安装迁移到 GoldLion
func OpenClaw(ocDir string, logger *slog.Logger) (*Result, error) {
	result := &Result{}

	// 检测 OpenClaw 安装
	if _, err := os.Stat(ocDir); err != nil {
		return nil, fmt.Errorf("OpenClaw 目录不存在: %s", ocDir)
	}

	wsDir := filepath.Join(ocDir, "workspace")
	if _, err := os.Stat(wsDir); err != nil {
		wsDir = ocDir // 有些安装 workspace 在根目录
	}

	logger.Info("检测到 OpenClaw 安装", "dir", ocDir)

	// 1. 迁移 MEMORY.md
	if err := migrateMemory(wsDir, result, logger); err != nil {
		logger.Warn("记忆迁移部分失败", "error", err)
	}

	// 2. 迁移 daily 记忆
	migrateDailyMemory(wsDir, result, logger)

	// 3. 迁移配置
	migrateConfig(ocDir, result, logger)

	// 4. 安全扫描
	securityScan(ocDir, result, logger)

	// 5. 迁移 Skills
	migrateSkills(wsDir, result, logger)

	return result, nil
}

func migrateMemory(wsDir string, result *Result, logger *slog.Logger) error {
	memFile := filepath.Join(wsDir, "MEMORY.md")
	data, err := os.ReadFile(memFile)
	if err != nil {
		return fmt.Errorf("MEMORY.md 不存在")
	}

	// 初始化 GoldLion 记忆存储
	store, err := memory.NewSQLiteStore(config.DataDir())
	if err != nil {
		return err
	}

	// 解析 MEMORY.md 并存入
	lines := strings.Split(string(data), "\n")
	var currentSection string
	var content strings.Builder
	count := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// 保存上一段
			if currentSection != "" && content.Len() > 0 {
				store.SaveMessage("openclaw-import", memory.Entry{
					Role:    "system",
					Content: fmt.Sprintf("[迁移自 OpenClaw] %s\n%s", currentSection, content.String()),
				})
				count++
			}
			currentSection = strings.TrimPrefix(line, "## ")
			content.Reset()
		} else {
			content.WriteString(line + "\n")
		}
	}
	// 最后一段
	if currentSection != "" && content.Len() > 0 {
		store.SaveMessage("openclaw-import", memory.Entry{
			Role:    "system",
			Content: fmt.Sprintf("[迁移自 OpenClaw] %s\n%s", currentSection, content.String()),
		})
		count++
	}

	result.MemoryEntries += count
	logger.Info("MEMORY.md 已迁移", "entries", count)
	return nil
}

func migrateDailyMemory(wsDir string, result *Result, logger *slog.Logger) {
	memDir := filepath.Join(wsDir, "memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		return
	}

	store, _ := memory.NewSQLiteStore(config.DataDir())
	count := 0

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
		if err != nil {
			continue
		}
		store.SaveMessage("openclaw-import", memory.Entry{
			Role:    "system",
			Content: fmt.Sprintf("[迁移自 OpenClaw daily/%s]\n%s", e.Name(), string(data)),
		})
		count++
	}

	result.MemoryEntries += count
	if count > 0 {
		logger.Info("日常记忆已迁移", "files", count)
	}
}

func migrateConfig(ocDir string, result *Result, logger *slog.Logger) {
	// 读取 openclaw.json
	cfgFile := filepath.Join(ocDir, "openclaw.json")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		result.Warnings = append(result.Warnings, "openclaw.json 不存在，跳过配置迁移")
		return
	}

	var ocCfg map[string]interface{}
	if err := json.Unmarshal(data, &ocCfg); err != nil {
		result.Warnings = append(result.Warnings, "openclaw.json 解析失败")
		return
	}

	glCfg := config.DefaultConfig()

	// 检测 Telegram 配置
	if plugins, ok := ocCfg["plugins"].(map[string]interface{}); ok {
		if _, hasTG := plugins["telegram"]; hasTG {
			glCfg.Channels.Telegram.Enabled = true
			logger.Info("检测到 Telegram 配置")
		}
	}

	config.Save(glCfg)
	result.ConfigMigrated = true
	logger.Info("配置已迁移")
}

func securityScan(ocDir string, result *Result, logger *slog.Logger) {
	// 扫描明文凭证
	patterns := []string{
		"openclaw.json",
		filepath.Join("workspace", "TOOLS.md"),
		filepath.Join("workspace", ".env"),
	}

	v, _ := vault.NewFileVault(config.ConfigDir())

	for _, pat := range patterns {
		path := filepath.Join(ocDir, pat)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)

		// 检测常见明文凭证模式
		scanPatterns := []struct {
			pattern string
			name    string
		}{
			{"ANTHROPIC_API_KEY", "Anthropic API Key"},
			{"OPENAI_API_KEY", "OpenAI API Key"},
			{"BOT_TOKEN", "Telegram Bot Token"},
			{"sk-", "API Key (sk- prefix)"},
		}

		for _, sp := range scanPatterns {
			if strings.Contains(content, sp.pattern) {
				// 尝试提取并加密存储
				extracted := extractValue(content, sp.pattern)
				if extracted != "" && v != nil {
					v.Set(sp.name, []byte(extracted))
					result.SecurityFixes = append(result.SecurityFixes,
						fmt.Sprintf("发现 %s 明文 → 已加密存入 Vault", sp.name))
				} else {
					result.SecurityFixes = append(result.SecurityFixes,
						fmt.Sprintf("⚠️ 发现 %s 明文于 %s", sp.name, pat))
				}
			}
		}
	}

	// 检查 gateway bind
	cfgFile := filepath.Join(ocDir, "openclaw.json")
	if data, err := os.ReadFile(cfgFile); err == nil {
		if strings.Contains(string(data), "0.0.0.0") {
			result.SecurityFixes = append(result.SecurityFixes,
				"OpenClaw Gateway 绑定 0.0.0.0 → GoldLion 默认 127.0.0.1")
		}
	}
}

func migrateSkills(wsDir string, result *Result, logger *slog.Logger) {
	skillDir := filepath.Join(wsDir, "skills")
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return
	}

	glSkillDir := filepath.Join(config.ConfigDir(), "skills")
	os.MkdirAll(glSkillDir, 0700)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// 检查是否有 SKILL.md
		skillMD := filepath.Join(skillDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			continue
		}

		// 复制到 GoldLion skills 目录
		destDir := filepath.Join(glSkillDir, e.Name())
		if err := copyDir(filepath.Join(skillDir, e.Name()), destDir); err != nil {
			result.SkillsSkipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skill %s 迁移失败: %v", e.Name(), err))
			continue
		}
		result.SkillsMigrated++
	}

	if result.SkillsMigrated > 0 {
		logger.Info("Skills 已迁移", "migrated", result.SkillsMigrated, "skipped", result.SkillsSkipped)
	}
}

// extractValue 简单提取 key=value 或 "key": "value"
func extractValue(content, key string) string {
	// JSON 格式: "key": "value"
	idx := strings.Index(content, key)
	if idx < 0 {
		return ""
	}

	rest := content[idx+len(key):]
	// 跳过分隔符
	rest = strings.TrimLeft(rest, `": =`)

	// 提取到引号或空白结束
	if len(rest) > 0 && rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end > 0 {
			return rest[1 : end+1]
		}
	}

	// 提取到空白结束
	scanner := bufio.NewScanner(strings.NewReader(rest))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	os.MkdirAll(dst, 0700)

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())

		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}
