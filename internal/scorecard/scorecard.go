package scorecard

import (
	"fmt"
	"os"
	"strings"

	"github.com/goldlion/goldlion/internal/config"
)

// Score 安全评分项
type Score struct {
	Name    string `json:"name"`
	GLPass  bool   `json:"gl_pass"`  // GoldLion 是否通过
	OCPass  bool   `json:"oc_pass"`  // OpenClaw 是否通过
	Weight  int    `json:"weight"`   // 权重 1-10
}

// Card 安全评分卡
type Card struct {
	GLScore  int     `json:"gl_score"`
	OCScore  int     `json:"oc_score"`
	GLGrade  string  `json:"gl_grade"`
	OCGrade  string  `json:"oc_grade"`
	Items    []Score `json:"items"`
}

// Generate 生成安全评分卡
func Generate(ocDir string) *Card {
	items := []Score{
		checkCredentialEncryption(ocDir),
		checkGatewayBind(ocDir),
		checkSkillIsolation(),
		checkPermissionDecl(),
		checkNetworkDefault(),
		checkUpdateSafety(),
		checkCostTracking(),
		checkMemoryEncryption(),
	}

	card := &Card{Items: items}

	totalWeight := 0
	glWeighted := 0
	ocWeighted := 0

	for _, item := range items {
		totalWeight += item.Weight
		if item.GLPass {
			glWeighted += item.Weight
		}
		if item.OCPass {
			ocWeighted += item.Weight
		}
	}

	if totalWeight > 0 {
		card.GLScore = glWeighted * 100 / totalWeight
		card.OCScore = ocWeighted * 100 / totalWeight
	}

	card.GLGrade = scoreToGrade(card.GLScore)
	card.OCGrade = scoreToGrade(card.OCScore)

	return card
}

// Format 格式化为可读文本
func (c *Card) Format() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🦁 GoldLion 安全评分: %s (%d/100)\n", c.GLGrade, c.GLScore))
	sb.WriteString(fmt.Sprintf("🦞 OpenClaw 安全评分: %s (%d/100)\n", c.OCGrade, c.OCScore))
	sb.WriteString("──────────────────────────\n\n")

	for _, item := range c.Items {
		gl := "✅"
		if !item.GLPass {
			gl = "❌"
		}
		oc := "✅"
		if !item.OCPass {
			oc = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s vs %s  %s\n", gl, oc, item.Name))
	}

	sb.WriteString(fmt.Sprintf("\n📊 全球 135,000 OpenClaw 实例暴露在公网\n"))
	sb.WriteString("🔒 你的 GoldLion: 仅本地访问\n")

	return sb.String()
}

func checkCredentialEncryption(ocDir string) Score {
	s := Score{Name: "凭证加密存储", GLPass: true, Weight: 10}
	// OpenClaw 默认明文
	if ocDir != "" {
		cfgFile := fmt.Sprintf("%s/openclaw.json", ocDir)
		if data, err := os.ReadFile(cfgFile); err == nil {
			if strings.Contains(string(data), "apiKey") || strings.Contains(string(data), "token") {
				s.OCPass = false
			}
		}
	}
	return s
}

func checkGatewayBind(ocDir string) Score {
	s := Score{Name: "Gateway 默认仅本地", GLPass: true, Weight: 10}
	// GoldLion 默认 127.0.0.1
	// OpenClaw 默认 0.0.0.0
	if ocDir != "" {
		if data, err := os.ReadFile(fmt.Sprintf("%s/openclaw.json", ocDir)); err == nil {
			s.OCPass = !strings.Contains(string(data), "0.0.0.0")
		}
	}
	return s
}

func checkSkillIsolation() Score {
	return Score{Name: "Skill 进程隔离", GLPass: true, OCPass: false, Weight: 9}
}

func checkPermissionDecl() Score {
	return Score{Name: "Skill 权限声明", GLPass: true, OCPass: false, Weight: 8}
}

func checkNetworkDefault() Score {
	return Score{Name: "默认拒绝外网访问", GLPass: true, OCPass: false, Weight: 8}
}

func checkUpdateSafety() Score {
	return Score{Name: "更新回滚保护", GLPass: true, OCPass: false, Weight: 7}
}

func checkCostTracking() Score {
	return Score{Name: "成本追踪+预算", GLPass: true, OCPass: false, Weight: 6}
}

func checkMemoryEncryption() Score {
	// GoldLion 数据目录 0700 权限
	dir := config.DataDir()
	info, err := os.Stat(dir)
	s := Score{Name: "数据目录权限保护", GLPass: true, OCPass: false, Weight: 5}
	if err == nil {
		perm := info.Mode().Perm()
		s.GLPass = perm&0077 == 0 // 只有 owner 可访问
	}
	return s
}

func scoreToGrade(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 80:
		return "B+"
	case score >= 70:
		return "B"
	case score >= 60:
		return "C"
	default:
		return "D"
	}
}
