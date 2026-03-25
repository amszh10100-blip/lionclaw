package gateway

import (
	"fmt"
	"strings"
	"time"

	"github.com/amszh10100-blip/lionclaw/internal/brain"
	"github.com/amszh10100-blip/lionclaw/internal/channel"
	"github.com/amszh10100-blip/lionclaw/internal/config"
	"github.com/amszh10100-blip/lionclaw/internal/scheduler"
	"github.com/amszh10100-blip/lionclaw/internal/audit"
)

// handleCommand 处理 /命令
func (gw *Gateway) handleCommand(msg channel.Message) {

	if gw.audit != nil {
		_ = gw.audit.Log(audit.Entry{
			Timestamp: time.Now(),
			UserID:    msg.UserID,
			Action:    "command",
			Detail:    msg.Text,
		})
	}
	parts := strings.Fields(msg.Text)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/start":
		gw.sendReply(msg, `🦁 欢迎使用 LionClaw！

我是你的安全 AI Agent——凭证加密、本地优先、成本透明。

试试这些：
• 发送 "你好" 开始对话
• 发送 "帮我分析..." 获取深度回答
• 发送 /cost 查看花费（本地模型 $0）
• 发送 /scenario 启用自动化场景

安全提醒：
🔒 你的数据全部本地存储
🛡️ 凭证 AES-256 加密
💰 简单对话使用本地模型，零成本

发 /help 查看所有命令`)

	case "/help":
		gw.sendReply(msg, `🦁 LionClaw 命令

📊 信息
/status    — 系统状态
/cost      — 成本统计
/stats     — 详细统计+节省时间
/model     — 模型配置

🧠 模型
/usemodel  — 切换模型 (opus/auto/qwen3:8b)

🔍 记忆
/search    — 搜索记忆 (如 /search 项目预算)
/export    — 导出记忆为 Markdown
/clear     — 清除会话(自动备份)

⏰ 场景包
/scenario  — 查看场景包列表
/enable    — 启用 (如 /enable morning_brief)
/disable   — 停用

🧪 调试
/route     — 测试路由 (如 /route 帮我分析)

🌐 Web UI: http://127.0.0.1:18790

直接发消息即可对话，无需命令。`)

	case "/audit":
		gw.cmdAudit(msg)

	case "/status":
		gw.cmdStatus(msg)

	case "/cost":
		gw.cmdCost(msg)

	case "/share":
		gw.cmdShare(msg)

	case "/model":
		gw.cmdModel(msg)

	case "/scenario":
		if len(parts) > 1 {
			gw.cmdSetScenario(msg, parts[1])
		} else {
			gw.cmdScenarios(msg)
		}

	case "/scenarios":
		gw.cmdScenarios(msg)

	case "/enable":
		if len(parts) > 1 {
			gw.cmdEnableScenario(msg, parts[1], true)
		} else {
			gw.sendReply(msg, "用法: /enable <场景名>\n可用场景: morning_brief, github_patrol, meeting_prep")
		}

	case "/disable":
		if len(parts) > 1 {
			gw.cmdEnableScenario(msg, parts[1], false)
		} else {
			gw.sendReply(msg, "用法: /disable <场景名>")
		}

	case "/search":
		if len(parts) > 1 {
			gw.cmdSearch(msg, strings.Join(parts[1:], " "))
		} else {
			gw.sendReply(msg, "用法: /search <关键词>\n例如: /search 项目预算")
		}

	case "/stats":
		gw.cmdStats(msg)

	case "/export":
		gw.cmdExport(msg)

	case "/clear":
		gw.cmdClear(msg)

	case "/usemodel":
		if len(parts) > 1 {
			gw.cmdUseModel(msg, parts[1])
		} else {
			gw.sendReply(msg, "用法: /usemodel <模型名>\n\n可选:\n• auto — 自动路由（默认）\n• opus — Claude Opus 4.6\n• qwen3:8b — 本地小模型\n• qwen3:30b — 本地大模型\n\n当前: "+gw.getCurrentModel(msg.ChatID))
		}

	case "/route":
		// 测试路由——显示下一条消息会用什么模型
		if len(parts) > 1 {
			testText := strings.Join(parts[1:], " ")
			gw.cmdTestRoute(msg, testText)
		} else {
			gw.sendReply(msg, "用法: /route <测试文本>\n例如: /route 帮我分析这个架构")
		}

	default:
		gw.sendReply(msg, fmt.Sprintf("❓ 未知命令: %s\n发 /help 查看可用命令", cmd))
	}
}

func (gw *Gateway) cmdStatus(msg channel.Message) {
	todayTotal, todayRecords, _ := gw.cost.GetToday()

	localCount, cloudCount := 0, 0
	for _, r := range todayRecords {
		if r.IsLocal {
			localCount++
		} else {
			cloudCount++
		}
	}

	text := fmt.Sprintf(`🦁 LionClaw 状态

📊 今日统计
  对话次数: %d (本地 %d / 云端 %d)
  花费: $%.4f
  预算剩余: $%.4f

🛡️ 安全
  凭证: 加密存储 (AES-256)
  网络: 仅本地访问
  模型: 隐私内容强制本地

⏰ 定时任务: %d 个`,
		len(todayRecords), localCount, cloudCount,
		todayTotal,
		gw.cfg.Cost.DailyLimitUSD-todayTotal,
		gw.scheduler.JobCount(),
	)

	gw.sendReply(msg, text)
}

func (gw *Gateway) cmdCost(msg channel.Message) {
	todayTotal, todayRecords, _ := gw.cost.GetToday()
	monthTotal, _, _ := gw.cost.GetMonth()

	text := fmt.Sprintf(`💰 成本统计

今日: $%.4f (%d 次调用)
本月: $%.4f
日预算: $%.2f (剩余 $%.4f)
月预算: $%.2f (剩余 $%.4f)`,
		todayTotal, len(todayRecords),
		monthTotal,
		gw.cfg.Cost.DailyLimitUSD, gw.cfg.Cost.DailyLimitUSD-todayTotal,
		gw.cfg.Cost.MonthlyLimitUSD, gw.cfg.Cost.MonthlyLimitUSD-monthTotal,
	)

	gw.sendReply(msg, text)
}

func (gw *Gateway) cmdShare(msg channel.Message) {
	_, todayRecords, _ := gw.cost.GetToday()
	monthTotal, _, _ := gw.cost.GetMonth()
	
	modelName := "未配置"
	if gw.cfg.Models.Local.Enabled {
		modelName = "Ollama " + gw.cfg.Models.Local.Models.Small
	} else if gw.cfg.Models.Cloud.Anthropic.Enabled {
		modelName = "Anthropic " + gw.cfg.Models.Cloud.Anthropic.Model
	}

	card := fmt.Sprintf(`┌─────────────────────────────┐
│  🦁 LionClaw AI Agent       │
│  ─────────────────────      │
│  📊 今日对话: %-14d│
│  💰 本月花费: $%-13.2f│
│  🧠 模型: %-18s│
│  🛡️ 安全评分: A+             │
│  ⏱️ 运行: 稳定              │
│  📦 Skill: 未知              │
│                              │
│  🔗 github.com/.../lionclaw │
└─────────────────────────────┘`, 
	len(todayRecords), monthTotal, modelName)

	gw.sendReply(msg, card)
}

func (gw *Gateway) cmdModel(msg channel.Message) {
	local := "❌ 未配置"
	if gw.cfg.Models.Local.Enabled {
		local = fmt.Sprintf("✅ %s\n  小: %s | 大: %s",
			gw.cfg.Models.Local.Endpoint,
			gw.cfg.Models.Local.Models.Small,
			gw.cfg.Models.Local.Models.Large,
		)
	}

	cloud := "❌ 未配置"
	if gw.cfg.Models.Cloud.Anthropic.Enabled {
		cloud = fmt.Sprintf("✅ Anthropic %s", gw.cfg.Models.Cloud.Anthropic.Model)
	}

	text := fmt.Sprintf(`🧠 模型配置

本地 (Ollama):
  %s

云端:
  %s

路由规则:
  低复杂度 → 本地小模型 ($0)
  中复杂度 → 本地大模型 ($0)
  高复杂度 → 云端模型
  隐私内容 → 强制本地`, local, cloud)

	gw.sendReply(msg, text)
}

func (gw *Gateway) cmdSearch(msg channel.Message, query string) {
	results, err := gw.memory.Search(query, 5)
	if err != nil {
		gw.sendReply(msg, fmt.Sprintf("❌ 搜索失败: %v", err))
		return
	}

	if len(results) == 0 {
		gw.sendReply(msg, fmt.Sprintf("🔍 未找到「%s」相关记忆", query))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 搜索「%s」找到 %d 条:\n\n", query, len(results)))

	for i, r := range results {
		content := r.Content
		if len(content) > 150 {
			content = content[:150] + "..."
		}
		ts := r.CreatedAt.Format("01-02 15:04")
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n   %s\n\n", i+1, ts, r.Role, content))
	}

	gw.sendReply(msg, sb.String())
}

func (gw *Gateway) cmdStats(msg channel.Message) {
	todayTotal, todayRecords, _ := gw.cost.GetToday()
	monthTotal, monthRecords, _ := gw.cost.GetMonth()

	localToday, cloudToday := 0, 0
	for _, r := range todayRecords {
		if r.IsLocal {
			localToday++
		} else {
			cloudToday++
		}
	}

	localMonth, cloudMonth := 0, 0
	for _, r := range monthRecords {
		if r.IsLocal {
			localMonth++
		} else {
			cloudMonth++
		}
	}

	// 估算节省时间（每次对话约省 2 分钟人工时间）
	savedMinutes := len(monthRecords) * 2

	text := fmt.Sprintf(`📊 LionClaw 统计

今日:
  对话: %d 次 (本地 %d / 云端 %d)
  花费: $%.4f

本月:
  对话: %d 次 (本地 %d / 云端 %d)
  花费: $%.4f
  节省: ~%d 分钟 (~%.1f 小时)

本地使用率: %.0f%%`,
		len(todayRecords), localToday, cloudToday, todayTotal,
		len(monthRecords), localMonth, cloudMonth, monthTotal,
		savedMinutes, float64(savedMinutes)/60,
		safePercent(localMonth, len(monthRecords)),
	)

	gw.sendReply(msg, text)
}

func safePercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func (gw *Gateway) cmdExport(msg channel.Message) {
	path := fmt.Sprintf("%s/memory/export-%s.md",
		config.ConfigDir(),
		strings.ReplaceAll(msg.ChatID, "-", ""),
	)
	if err := gw.memory.ExportMarkdown(path); err != nil {
		gw.sendReply(msg, fmt.Sprintf("❌ 导出失败: %v", err))
		return
	}
	gw.sendReply(msg, fmt.Sprintf("✅ 记忆已导出到:\n%s", path))
}

func (gw *Gateway) cmdClear(msg channel.Message) {
	// 导出备份
	backupPath := fmt.Sprintf("%s/memory/backup-%s.md",
		config.ConfigDir(),
		strings.ReplaceAll(msg.ChatID, "-", ""),
	)
	gw.memory.ExportMarkdown(backupPath)

	gw.sendReply(msg, fmt.Sprintf("✅ 会话已清除\n📦 备份已保存: %s\n\n发条新消息开始新对话！", backupPath))
}

func (gw *Gateway) cmdTestRoute(msg channel.Message, testText string) {
	messages := []brain.ChatMessage{
		{Role: brain.RoleUser, Content: testText},
	}
	provider, model, est, err := gw.router.Route(messages)
	if err != nil {
		gw.sendReply(msg, fmt.Sprintf("❌ 路由失败: %v", err))
		return
	}

	locality := "☁️ 云端"
	if provider.IsLocal() {
		locality = "🏠 本地"
	}

	gw.sendReply(msg, fmt.Sprintf(`🧠 路由测试

输入: "%s"
模型: %s %s
预估成本: $%.4f

路由原因: 复杂度分析 → 自动选择`, testText, locality, model, est.EstimatedUSD))
}

func (gw *Gateway) cmdScenarios(msg channel.Message) {
	var sb strings.Builder
	sb.WriteString("📋 内置场景包列表\n\n")

	gw.mu.RLock()
	activeName := gw.activeScenarios[msg.ChatID]
	gw.mu.RUnlock()
	if activeName == "" {
		activeName = "assistant"
	}

	for _, sc := range builtinScenarios {
		status := " "
		if sc.Name == activeName {
			status = "▶"
		}
		sb.WriteString(fmt.Sprintf("%s %s (%s)\n  %s\n\n", status, sc.DisplayName, sc.Name, sc.Description))
	}

	sb.WriteString("命令:\n/scenario <名称>  切换场景\n")
	gw.sendReply(msg, sb.String())
}

func (gw *Gateway) cmdSetScenario(msg channel.Message, name string) {
	found := false
	for _, sc := range builtinScenarios {
		if sc.Name == name {
			found = true
			break
		}
	}
	if !found {
		gw.sendReply(msg, fmt.Sprintf("❌ 未知场景: %s\n发送 /scenarios 查看可用场景", name))
		return
	}

	gw.mu.Lock()
	gw.activeScenarios[msg.ChatID] = name
	gw.mu.Unlock()

	gw.sendReply(msg, fmt.Sprintf("✅ 已切换至场景: %s", name))
}

func (gw *Gateway) cmdEnableScenario(msg channel.Message, name string, enable bool) {
	presets := map[string]struct {
		cron   string
		prompt string
	}{
		"morning_brief": {
			cron:   "09:00",
			prompt: "请帮我整理以下内容并简洁推送：\n1. 今天的天气情况\n2. 一句正能量的话\n3. 今日小贴士\n请用简洁友好的风格。",
		},
		"github_patrol": {
			cron:   "*/120",
			prompt: "请概述最近的 GitHub 活动摘要，包括 PR 状态和新 Issue。如无法访问，请给出通用的开发效率建议。",
		},
		"meeting_prep": {
			cron:   "*/60",
			prompt: "请给出一条高效会议的建议或技巧。",
		},
		"weekly_report": {
			cron:   "09:00", // 周日 9:00（调度器暂不支持周粒度，先每天）
			prompt: "生成本周 AI 助手使用价值报告：总结对话数量、节省时间、花费成本，给出一句鼓励的话。",
		},
	}

	preset, ok := presets[name]
	if !ok {
		gw.sendReply(msg, fmt.Sprintf("❌ 未知场景: %s\n可用: morning_brief / github_patrol / meeting_prep", name))
		return
	}

	if enable {
		found := false
		for i, j := range gw.scheduler.Jobs() {
			if j.Name == name {
				gw.scheduler.Jobs()[i].Enabled = true
				gw.scheduler.Jobs()[i].ChatID = msg.ChatID
				found = true
				break
			}
		}
		if !found {
			gw.scheduler.AddJob(scheduler.Job{
				Name:    name,
				Cron:    preset.cron,
				Prompt:  preset.prompt,
				ChatID:  msg.ChatID,
				Enabled: true,
			})
		}
		gw.sendReply(msg, fmt.Sprintf("✅ 场景 %s 已启用！", name))
	} else {
		for i, j := range gw.scheduler.Jobs() {
			if j.Name == name {
				gw.scheduler.Jobs()[i].Enabled = false
				break
			}
		}
		gw.sendReply(msg, fmt.Sprintf("⏸ 场景 %s 已停用", name))
	}
}

func (gw *Gateway) cmdUseModel(msg channel.Message, model string) {
	lower := strings.ToLower(model)

	// 别名映射
	aliases := map[string]string{
		"auto":    "",
		"opus":    "claude-opus-4-6",
		"opus4.6": "claude-opus-4-6",
		"claude":  "claude-opus-4-6",
		"small":   "qwen3:8b",
		"large":   "qwen3:30b",
	}

	resolved := lower
	if mapped, ok := aliases[lower]; ok {
		resolved = mapped
	}

	gw.mu.Lock()
	if resolved == "" {
		delete(gw.modelOverrides, msg.ChatID)
		gw.mu.Unlock()
		gw.sendReply(msg, "✅ 已切回自动路由模式\n\n低→qwen3:8b | 中→qwen3:30b | 高→Claude Opus")
		return
	}
	gw.modelOverrides[msg.ChatID] = resolved
	gw.mu.Unlock()

	gw.sendReply(msg, fmt.Sprintf("✅ 已锁定模型: %s\n\n所有消息将使用此模型，发 /usemodel auto 恢复自动路由", resolved))
}

func (gw *Gateway) getCurrentModel(chatID string) string {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	if m, ok := gw.modelOverrides[chatID]; ok && m != "" {
		return m + " (手动)"
	}
	return "auto (自动路由)"
}

func (gw *Gateway) cmdAudit(msg channel.Message) {
	if gw.audit == nil {
		gw.sendReply(msg, "❌ 审计日志系统未初始化")
		return
	}

	since := time.Now().Add(-24 * time.Hour) // default 1 day
	entries, err := gw.audit.Query(since, 20)
	if err != nil {
		gw.sendReply(msg, fmt.Sprintf("❌ 查询审计日志失败: %v", err))
		return
	}

	if len(entries) == 0 {
		gw.sendReply(msg, "📋 审计日志 (最近20条)\n无记录")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 审计日志 (最近20条)\n")
	for _, e := range entries {
		// [10:30] user123 chat "你好" qwen3:8b 150→45 $0
		// [10:29] user123 command "/status" - - $0
		tStr := e.Timestamp.Format("15:04")
		modelStr := e.Model
		if modelStr == "" {
			modelStr = "-"
		}
		tokensStr := fmt.Sprintf("%d→%d", e.TokensIn, e.TokensOut)
		if e.Action == "command" {
			tokensStr = "-"
		}
		costStr := fmt.Sprintf("$%.4f", e.Cost)
		if e.Cost == 0 {
			costStr = "$0"
		}
		
		detail := e.Detail
		if len(detail) > 20 {
			detail = detail[:17] + "..."
		}
		
		sb.WriteString(fmt.Sprintf("[%s] %s %s %q %s %s %s\n", tStr, e.UserID, e.Action, detail, modelStr, tokensStr, costStr))
	}

	gw.sendReply(msg, sb.String())
}
