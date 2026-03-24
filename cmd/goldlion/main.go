package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"log/slog"
	"path/filepath"

	"github.com/goldlion/goldlion/internal/brain"
	"github.com/goldlion/goldlion/internal/config"
	"github.com/goldlion/goldlion/internal/gateway"
	"github.com/goldlion/goldlion/internal/migrate"
	"github.com/goldlion/goldlion/internal/scorecard"
	"github.com/goldlion/goldlion/internal/skill"
	"github.com/goldlion/goldlion/internal/vault"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		cmdStart()
	case "setup":
		cmdSetup()
	case "status":
		cmdStatus()
	case "version":
		fmt.Printf("goldlion v%s\n", version)
	case "skill":
		cmdSkill()
	case "vault":
		cmdVault()
	case "cost":
		cmdCost()
	case "migrate":
		cmdMigrate()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`🦁 GoldLion v%s — 安全的个人 AI Agent

Usage:
  goldlion <command>

Commands:
  start     启动 Gateway
  setup     交互式配置引导
  status    查看运行状态
  skill     Skill 管理 (create/list/audit)
  vault     管理加密凭证 (set/list/delete)
  cost      查看成本统计
  migrate   从 OpenClaw 迁移
  version   显示版本
`, version)
}

func cmdStart() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 加载配置失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "   运行 `goldlion setup` 进行初始配置\n")
		os.Exit(1)
	}

	gw, err := gateway.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 初始化 Gateway 失败: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n🦁 正在关闭 GoldLion...")
		cancel()
	}()

	fmt.Printf("🦁 GoldLion v%s 启动中...\n", version)
	fmt.Printf("   绑定: %s:%d\n", cfg.Security.Bind, cfg.Security.Port)

	if err := gw.Run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "❌ Gateway 异常退出: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🦁 GoldLion 已停止")
}

func cmdSetup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("🦁 GoldLion 交互式配置\n")

	cfg := config.DefaultConfig()

	// Step 1: Telegram
	fmt.Print("① Telegram Bot Token (从 @BotFather 获取): ")
	tgToken, _ := reader.ReadString('\n')
	tgToken = strings.TrimSpace(tgToken)

	if tgToken != "" {
		cfg.Channels.Telegram.Enabled = true

		// 存入 Vault
		v, err := vault.NewFileVault(config.ConfigDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Vault 初始化失败: %v\n", err)
			os.Exit(1)
		}
		if err := v.Set("TELEGRAM_BOT_TOKEN", []byte(tgToken)); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 保存 Token 失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("   ✅ Token 已加密存储到 Vault")
	}

	// Step 2: 云端模型 API Key（可选）
	v2, _ := vault.NewFileVault(config.ConfigDir())

	fmt.Print("\n② Anthropic API Key (可选，回车跳过): ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		cfg.Models.Cloud.Anthropic.Enabled = true
		cfg.Models.Cloud.Anthropic.Model = "claude-opus-4-6"
		v2.Set("ANTHROPIC_API_KEY", []byte(apiKey))
		fmt.Println("   ✅ Anthropic Key 已加密存储")
	}

	fmt.Print("   OpenAI API Key (可选，回车跳过): ")
	oaiKey, _ := reader.ReadString('\n')
	oaiKey = strings.TrimSpace(oaiKey)
	if oaiKey != "" {
		cfg.Models.Cloud.OpenAI.Enabled = true
		cfg.Models.Cloud.OpenAI.Model = "gpt-5.1"
		v2.Set("OPENAI_API_KEY", []byte(oaiKey))
		fmt.Println("   ✅ OpenAI Key 已加密存储")
	}

	// Step 3: 本地模型
	fmt.Println("\n③ 本地模型 (Ollama)")
	fmt.Printf("   端点: %s\n", cfg.Models.Local.Endpoint)
	fmt.Printf("   小模型: %s | 大模型: %s\n", cfg.Models.Local.Models.Small, cfg.Models.Local.Models.Large)
	fmt.Println("   ✅ 默认配置已就绪")

	// 保存配置
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ 配置已保存到 %s\n", config.ConfigPath())
	fmt.Println("\n🚀 运行 `goldlion start` 启动 Gateway")
}

func cmdStatus() {
	fmt.Println("🦁 GoldLion 状态")
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "未配置。运行 `goldlion setup`\n")
		return
	}
	fmt.Printf("   Telegram: %v\n", cfg.Channels.Telegram.Enabled)
	fmt.Printf("   本地模型: %v (%s)\n", cfg.Models.Local.Enabled, cfg.Models.Local.Endpoint)
	fmt.Printf("   云端模型: Anthropic=%v\n", cfg.Models.Cloud.Anthropic.Enabled)
	fmt.Printf("   日预算: $%.2f | 月预算: $%.2f\n", cfg.Cost.DailyLimitUSD, cfg.Cost.MonthlyLimitUSD)
	fmt.Printf("   绑定: %s:%d\n", cfg.Security.Bind, cfg.Security.Port)
}

func cmdVault() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: goldlion vault <set|get|list|delete> [key] [value]")
		return
	}

	v, err := vault.NewFileVault(config.ConfigDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Vault 初始化失败: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[2] {
	case "set":
		if len(os.Args) < 5 {
			fmt.Println("Usage: goldlion vault set <key> <value>")
			return
		}
		if err := v.Set(os.Args[3], []byte(os.Args[4])); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ %s 已加密存储\n", os.Args[3])

	case "list":
		keys, _ := v.List()
		if len(keys) == 0 {
			fmt.Println("(空)")
			return
		}
		for _, k := range keys {
			fmt.Printf("  🔑 %s\n", k)
		}

	case "delete":
		if len(os.Args) < 4 {
			fmt.Println("Usage: goldlion vault delete <key>")
			return
		}
		if err := v.Delete(os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ %s 已删除\n", os.Args[3])

	default:
		fmt.Println("Usage: goldlion vault <set|list|delete>")
	}
}

func cmdSkill() {
	if len(os.Args) < 3 {
		fmt.Println(`🦁 Skill 管理

Usage: goldlion skill <command>

Commands:
  create <name>   创建新 Skill 脚手架
  list            列出已安装 Skill
  audit <path>    安全审计 Skill`)
		return
	}

	skillDir := filepath.Join(config.ConfigDir(), "skills")

	switch os.Args[2] {
	case "create":
		if len(os.Args) < 4 {
			fmt.Println("用法: goldlion skill create <name>")
			return
		}
		name := os.Args[3]
		if err := skill.Create(skillDir, name); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Skill '%s' 已创建:\n", name)
		fmt.Printf("   %s/%s/\n", skillDir, name)
		fmt.Printf("   ├── skill.yaml   (配置)\n")
		fmt.Printf("   ├── run.sh       (入口)\n")
		fmt.Printf("   ├── test.sh      (测试)\n")
		fmt.Printf("   └── README.md    (文档)\n")
		fmt.Printf("\n编辑 run.sh 实现你的逻辑，然后 goldlion skill audit %s/%s\n", skillDir, name)

	case "list":
		entries, err := os.ReadDir(skillDir)
		if err != nil {
			fmt.Println("(无已安装 Skill)")
			return
		}
		fmt.Println("🦁 已安装 Skills:")
		count := 0
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			mPath := filepath.Join(skillDir, e.Name(), "skill.yaml")
			if _, err := os.Stat(mPath); err != nil {
				continue
			}
			fmt.Printf("  📦 %s\n", e.Name())
			count++
		}
		if count == 0 {
			fmt.Println("  (空)")
		}
		fmt.Printf("\n共 %d 个 Skill\n", count)

	case "audit":
		if len(os.Args) < 4 {
			fmt.Println("用法: goldlion skill audit <path>")
			return
		}
		results, err := skill.Audit(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Println("🛡️ Skill 安全审计:")
		for _, r := range results {
			icon := "✅"
			if r.Status == "warn" {
				icon = "⚠️"
			} else if r.Status == "error" {
				icon = "❌"
			}
			fmt.Printf("  %s %s: %s\n", icon, r.Check, r.Detail)
		}

	default:
		fmt.Printf("未知命令: skill %s\n", os.Args[2])
	}
}

func cmdCost() {
	cfg := config.DefaultConfig()
	tracker, err := brain.NewSQLiteCostTracker(config.DataDir(), cfg.Cost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 无法读取成本数据: %v\n", err)
		os.Exit(1)
	}

	todayTotal, todayRecords, _ := tracker.GetToday()
	monthTotal, _, _ := tracker.GetMonth()

	fmt.Println("🦁 成本统计")
	fmt.Println("───────────────────────────")
	fmt.Printf("  今日花费: $%.4f (%d 次调用)\n", todayTotal, len(todayRecords))
	fmt.Printf("  本月花费: $%.4f\n", monthTotal)
	fmt.Printf("  日预算:   $%.2f (剩余 $%.4f)\n", cfg.Cost.DailyLimitUSD, cfg.Cost.DailyLimitUSD-todayTotal)
	fmt.Printf("  月预算:   $%.2f (剩余 $%.4f)\n", cfg.Cost.MonthlyLimitUSD, cfg.Cost.MonthlyLimitUSD-monthTotal)

	if len(todayRecords) > 0 {
		fmt.Println("\n  今日明细:")
		localCount, cloudCount := 0, 0
		for _, r := range todayRecords {
			if r.IsLocal {
				localCount++
			} else {
				cloudCount++
			}
		}
		fmt.Printf("    本地调用: %d 次 ($0)\n", localCount)
		fmt.Printf("    云端调用: %d 次 ($%.4f)\n", cloudCount, todayTotal)
	}
}

func cmdMigrate() {
	logger := slog.Default()

	// 检测 OpenClaw 目录
	ocDir := os.Getenv("HOME") + "/.openclaw"
	if len(os.Args) > 2 {
		ocDir = os.Args[2]
	}

	fmt.Println("🦁 OpenClaw → GoldLion 迁移工具\n")
	fmt.Printf("   源目录: %s\n\n", ocDir)

	result, err := migrate.OpenClaw(ocDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 迁移失败: %v\n", err)
		os.Exit(1)
	}

	// 显示结果
	fmt.Println("📋 迁移结果:")
	fmt.Printf("   记忆条目: %d 条\n", result.MemoryEntries)
	fmt.Printf("   Skills: %d 已迁移, %d 跳过\n", result.SkillsMigrated, result.SkillsSkipped)
	fmt.Printf("   配置: %v\n", result.ConfigMigrated)

	if len(result.SecurityFixes) > 0 {
		fmt.Println("\n🛡️ 安全修复:")
		for _, fix := range result.SecurityFixes {
			fmt.Printf("   • %s\n", fix)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️ 注意:")
		for _, w := range result.Warnings {
			fmt.Printf("   • %s\n", w)
		}
	}

	// 生成安全评分卡
	fmt.Println("\n" + strings.Repeat("─", 40))
	card := scorecard.Generate(ocDir)
	fmt.Println(card.Format())

	fmt.Println("✅ 迁移完成！运行 `goldlion start` 启动")
}
