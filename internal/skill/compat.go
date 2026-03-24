package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConvertOpenClawSkill 将 OpenClaw SKILL.md 转换为 LionClaw skill.yaml
func ConvertOpenClawSkill(ocSkillDir string) (*Manifest, error) {
	skillMD := filepath.Join(ocSkillDir, "SKILL.md")
	data, err := os.ReadFile(skillMD)
	if err != nil {
		return nil, fmt.Errorf("SKILL.md 不存在: %w", err)
	}

	content := string(data)

	// 解析 YAML frontmatter
	m := &Manifest{
		Name:    filepath.Base(ocSkillDir),
		Version: "1.0.0-compat",
		Permissions: Permissions{
			Filesystem: "none",
		},
		Entrypoint: "SKILL.md", // OpenClaw skills 是 prompt-based
	}

	// 提取 frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content[3:], "---", 2)
		if len(parts) == 2 {
			frontmatter := parts[0]
			parseFrontmatter(frontmatter, m)
		}
	}

	// 推断权限
	lowerContent := strings.ToLower(content)
	if strings.Contains(lowerContent, "web_search") || strings.Contains(lowerContent, "web_fetch") {
		m.Permissions.Network = append(m.Permissions.Network, "*")
	}
	if strings.Contains(lowerContent, "exec") || strings.Contains(lowerContent, "shell") {
		m.Permissions.Filesystem = "read"
	}
	if strings.Contains(lowerContent, "write") || strings.Contains(lowerContent, "edit") {
		m.Permissions.Filesystem = "write"
	}

	return m, nil
}

// BatchConvert 批量转换 OpenClaw Skills
func BatchConvert(ocSkillsDir, glSkillsDir string) (converted, skipped int, errors []string) {
	entries, err := os.ReadDir(ocSkillsDir)
	if err != nil {
		return 0, 0, []string{err.Error()}
	}

	os.MkdirAll(glSkillsDir, 0700)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		srcDir := filepath.Join(ocSkillsDir, e.Name())
		skillMD := filepath.Join(srcDir, "SKILL.md")

		if _, err := os.Stat(skillMD); err != nil {
			skipped++
			continue
		}

		m, err := ConvertOpenClawSkill(srcDir)
		if err != nil {
			skipped++
			errors = append(errors, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}

		// 写入 LionClaw 格式
		destDir := filepath.Join(glSkillsDir, e.Name())
		os.MkdirAll(destDir, 0700)

		// 生成 skill.yaml
		yamlData, _ := yaml.Marshal(m)
		os.WriteFile(filepath.Join(destDir, "skill.yaml"), yamlData, 0600)

		// 复制 SKILL.md 作为 prompt
		ocData, _ := os.ReadFile(skillMD)
		os.WriteFile(filepath.Join(destDir, "SKILL.md"), ocData, 0600)

		// 复制其他文件（scripts/, references/ 等）
		copySubDirs(srcDir, destDir)

		converted++
	}

	return
}

func parseFrontmatter(fm string, m *Manifest) {
	var meta map[string]interface{}
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return
	}

	if name, ok := meta["name"].(string); ok {
		m.Name = name
	}
	if desc, ok := meta["description"].(string); ok {
		m.Description = desc
	}
	if ver, ok := meta["version"].(string); ok {
		m.Version = ver
	}
}

func copySubDirs(src, dst string) {
	subDirs := []string{"scripts", "references", "templates", "assets"}
	for _, sub := range subDirs {
		srcSub := filepath.Join(src, sub)
		if _, err := os.Stat(srcSub); err == nil {
			dstSub := filepath.Join(dst, sub)
			copyDirRecursive(srcSub, dstSub)
		}
	}
}

func copyDirRecursive(src, dst string) {
	os.MkdirAll(dst, 0700)
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDirRecursive(s, d)
		} else {
			data, _ := os.ReadFile(s)
			os.WriteFile(d, data, 0600)
		}
	}
}
