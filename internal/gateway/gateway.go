package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lionclaw/lionclaw/internal/brain"
	"github.com/lionclaw/lionclaw/internal/channel"
	channeltg "github.com/lionclaw/lionclaw/internal/channel/telegram"
	"github.com/lionclaw/lionclaw/internal/config"
	"github.com/lionclaw/lionclaw/internal/memory"
	"github.com/lionclaw/lionclaw/internal/scheduler"
	"github.com/lionclaw/lionclaw/internal/vault"
	"github.com/lionclaw/lionclaw/internal/webui"
	"github.com/lionclaw/lionclaw/internal/audit"
)

// Gateway 是 LionClaw 的核心网关
type Gateway struct {
	cfg             *config.Config
	channels        []channel.Channel
	router          *brain.DefaultRouter
	cost            brain.CostTracker
	memory          memory.Store
	vault           vault.Vault
	scheduler       *scheduler.Scheduler
	audit           *audit.Logger
	logger          *slog.Logger
	mu              sync.RWMutex
	rateLimit       map[string][]time.Time // 用户速率限制
	rateMu          sync.Mutex
	activeScenarios map[string]string      // 用户活跃场景
}

// New 创建新的 Gateway 实例
func New(cfg *config.Config) (*Gateway, error) {
	logger := slog.Default()

	// 初始化记忆存储
	store, err := memory.NewSQLiteStore(config.DataDir())
	if err != nil {
		return nil, fmt.Errorf("初始化记忆存储失败: %w", err)
	}

	// 初始化成本追踪
	costTracker, err := brain.NewSQLiteCostTracker(config.DataDir(), cfg.Cost)
	if err != nil {
		return nil, fmt.Errorf("初始化成本追踪失败: %w", err)
	}

	// 初始化凭证 Vault
	v, err := vault.NewFileVault(config.ConfigDir())
	if err != nil {
		return nil, fmt.Errorf("初始化 Vault 失败: %w", err)
	}

	// 初始化模型路由
	modelRouter, err := brain.NewRouter(cfg, costTracker, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化模型路由失败: %w", err)
	}

	// 初始化审计日志
	auditLogger, err := audit.NewLogger(config.DataDir())
	if err != nil {
		return nil, fmt.Errorf("初始化审计日志失败: %w", err)
	}

	// 配置云端模型（优先 Anthropic，其次 OpenAI）
	if cfg.Models.Cloud.Anthropic.Enabled && v.Has("ANTHROPIC_API_KEY") {
		apiKey, _ := v.Get("ANTHROPIC_API_KEY")
		provider := brain.NewAnthropicProvider(string(apiKey))
		modelRouter.SetCloudProvider(provider, cfg.Models.Cloud.Anthropic.Model)
		logger.Info("云端模型已激活", "provider", "anthropic", "model", cfg.Models.Cloud.Anthropic.Model)
	} else if cfg.Models.Cloud.OpenAI.Enabled && v.Has("OPENAI_API_KEY") {
		apiKey, _ := v.Get("OPENAI_API_KEY")
		provider := brain.NewOpenAIProvider(string(apiKey))
		modelRouter.SetCloudProvider(provider, cfg.Models.Cloud.OpenAI.Model)
		logger.Info("云端模型已激活", "provider", "openai", "model", cfg.Models.Cloud.OpenAI.Model)
	}

	// 初始化调度器
	sched := scheduler.New(logger)

	gw := &Gateway{
		cfg:             cfg,
		router:          modelRouter,
		cost:            costTracker,
		memory:          store,
		vault:           v,
		scheduler:       sched,
		audit:           auditLogger,
		logger:          logger,
		rateLimit:       make(map[string][]time.Time),
		activeScenarios: make(map[string]string),
	}

	return gw, nil
}

// Run 启动 Gateway 主循环
func (gw *Gateway) Run(ctx context.Context) error {
	// 注册渠道
	if err := gw.initChannels(); err != nil {
		return fmt.Errorf("初始化渠道失败: %w", err)
	}

	// 启动所有渠道
	for _, ch := range gw.channels {
		ch.OnMessage(gw.handleMessage)
		if err := ch.Start(ctx); err != nil {
			return fmt.Errorf("启动渠道 %s 失败: %w", ch.Name(), err)
		}
		gw.logger.Info("渠道已启动", "channel", ch.Name())
	}

	// 启动 Web UI
	web := webui.New(gw.cfg, gw.cost, gw.memory, func() string {
		gw.mu.RLock()
		defer gw.mu.RUnlock()
		for _, sc := range gw.activeScenarios {
			return sc // return any active
		}
		return "assistant"
	}, gw.logger)
	go func() {
		if err := web.Start(ctx); err != nil {
			gw.logger.Error("Web UI 启动失败", "error", err)
		}
	}()

	// 加载场景包
	gw.loadScenarios()

	// 启动调度器
	gw.scheduler.SetHandler(gw.handleScheduledJob)
	go gw.scheduler.Start(ctx)

	gw.logger.Info("🦁 LionClaw Gateway 就绪",
		"channels", len(gw.channels),
		"scheduled_jobs", gw.scheduler.JobCount(),
	)

	// 阻塞直到 context 取消
	<-ctx.Done()

	// 优雅关闭
	for _, ch := range gw.channels {
		if err := ch.Stop(); err != nil {
			gw.logger.Error("关闭渠道失败", "channel", ch.Name(), "error", err)
		}
	}

	return nil
}

// handleMessage 处理收到的消息
func (gw *Gateway) handleMessage(msg channel.Message) {
	ctx := context.Background()

	gw.logger.Info("收到消息",
		"channel", "telegram",
		"user", msg.UserID,
		"text_len", len(msg.Text),
	)

	// 0. 速率限制（每用户每分钟 10 条）
	if !gw.checkRateLimit(msg.UserID, 10, time.Minute) {
		gw.sendReply(msg, "⚠️ 发送太快了，请稍后再试")
		return
	}

	// 0.1 命令处理
	if strings.HasPrefix(msg.Text, "/") {
		gw.handleCommand(msg)
		return
	}

	// 0.2 发送 typing 状态
	for _, ch := range gw.channels {
		if tg, ok := ch.(interface{ SendTyping(string) }); ok {
			tg.SendTyping(msg.ChatID)
		}
	}

	// 记录用户 chatID（供场景包使用）
	gw.mu.Lock()
	for i := range gw.scheduler.Jobs() {
		if gw.scheduler.Jobs()[i].ChatID == "" {
			gw.scheduler.Jobs()[i].ChatID = msg.ChatID
		}
	}
	gw.mu.Unlock()

	// 1. 检查成本预算
	allowed, remaining, err := gw.cost.CheckBudget(0.01) // 预估最低成本
	if err != nil {
		gw.logger.Error("检查预算失败", "error", err)
	}
	if !allowed {
		gw.sendReply(msg, fmt.Sprintf("⚠️ 今日预算已用完（剩余 $%.4f）。明天再来！", remaining))
		return
	}

	// 2. 加载会话上下文
	history, err := gw.memory.GetHistory(msg.ChatID, 20)
	if err != nil {
		gw.logger.Error("加载历史失败", "error", err)
	}
	if len(history) > 0 {
		gw.logger.Info("加载对话历史", "count", len(history))
	}

	// 3. 构建消息列表
	messages := gw.buildMessages(msg.ChatID, history, msg.Text)

	// 4. 模型路由
	provider, model, est, err := gw.router.Route(messages)
	if err != nil {
		gw.logger.Error("模型路由失败", "error", err)
		gw.sendReply(msg, "❌ 模型路由失败，请检查配置")
		return
	}

	gw.logger.Info("路由决策",
		"model", model,
		"local", provider.IsLocal(),
		"est_cost", est.EstimatedUSD,
	)

	// 5. 调用 LLM
	req := brain.ChatRequest{
		Messages: messages,
		Model:    model,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		gw.logger.Error("LLM 调用失败", "error", err)
		gw.sendReply(msg, fmt.Sprintf("❌ AI 调用失败: %v", err))
		return
	}

	// 6. 记录成本
	if err := gw.cost.Record(brain.CostRecord{
		Model:        resp.Model,
		IsLocal:      provider.IsLocal(),
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
	}); err != nil {
		gw.logger.Error("记录成本失败", "error", err)
	}

	// 7. 保存到记忆
	gw.saveToMemory(msg.ChatID, "user", msg.Text, 0, "", 0)
	gw.saveToMemory(msg.ChatID, "assistant", resp.Content, resp.InputTokens+resp.OutputTokens, resp.Model, resp.CostUSD)

	// 8. 记录审计日志
	detail := msg.Text
	if len(detail) > 100 {
		detail = detail[:97] + "..."
	}
	if gw.audit != nil {
		_ = gw.audit.Log(audit.Entry{
			Timestamp: time.Now(),
			UserID:    msg.UserID,
			Action:    "chat",
			Detail:    detail,
			Model:     resp.Model,
			TokensIn:  resp.InputTokens,
			TokensOut: resp.OutputTokens,
			Cost:      resp.CostUSD,
		})
	}

	// 8. 预算预警检查
	gw.checkBudgetWarning(msg, resp.CostUSD)

	// 9. 检查上下文大小，需要时触发压缩
	go gw.maybeCompress(msg.ChatID)

	// 9. 构建回复（附带模型信息）
	costLabel := fmt.Sprintf("$%.4f", resp.CostUSD)
	if provider.IsLocal() {
		costLabel = "$0"
	}
	header := fmt.Sprintf("⚡ %s | %s", resp.Model, costLabel)
	reply := fmt.Sprintf("%s\n\n%s", header, resp.Content)

	// 9. 发送回复
	gw.sendReply(msg, reply)
}

func (gw *Gateway) buildMessages(chatID string, history []memory.Entry, userText string) []brain.ChatMessage {
	sysPrompt := "你是 LionClaw 🦁，一个高效、直接的 AI 助手。先给答案再解释。全中文回复。" // fallback
	gw.mu.RLock()
	activeName := gw.activeScenarios[chatID]
	gw.mu.RUnlock()

	if activeName == "" {
		activeName = "assistant"
	}

	for _, sc := range builtinScenarios {
		if sc.Name == activeName {
			sysPrompt = sc.SystemPrompt
			break
		}
	}

	messages := []brain.ChatMessage{
		{
			Role:    brain.RoleSystem,
			Content: sysPrompt,
		},
	}

	// 附加历史上下文
	for _, h := range history {
		messages = append(messages, brain.ChatMessage{
			Role:    brain.Role(h.Role),
			Content: h.Content,
		})
	}

	// 当前用户消息
	messages = append(messages, brain.ChatMessage{
		Role:    brain.RoleUser,
		Content: userText,
	})

	return messages
}

func (gw *Gateway) saveToMemory(sessionID, role, content string, tokens int, model string, cost float64) {
	if err := gw.memory.SaveMessage(sessionID, memory.Entry{
		Role:    role,
		Content: content,
		Tokens:  tokens,
	}); err != nil {
		gw.logger.Error("保存记忆失败", "error", err)
	}
}

func (gw *Gateway) sendReply(msg channel.Message, text string) {
	// Telegram 单条消息限制 4096 字符
	const maxLen = 4000
	if len(text) > maxLen {
		text = text[:maxLen] + "\n\n... (截断)"
	}

	for _, ch := range gw.channels {
		if err := ch.Send(msg.ChatID, text, nil); err != nil {
			gw.logger.Error("发送回复失败", "channel", ch.Name(), "error", err, "text_len", len(text))
			// 尝试发送纯文本错误提示
			ch.Send(msg.ChatID, "❌ 发送回复失败，请重试", nil)
		} else {
			gw.logger.Info("回复已发送", "channel", ch.Name(), "text_len", len(text))
		}
	}
}

// checkRateLimit 速率限制
func (gw *Gateway) checkRateLimit(userID string, maxPerWindow int, window time.Duration) bool {
	gw.rateMu.Lock()
	defer gw.rateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// 清理过期记录
	times := gw.rateLimit[userID]
	var valid []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= maxPerWindow {
		return false
	}

	gw.rateLimit[userID] = append(valid, now)
	return true
}

// checkBudgetWarning 预算预警
func (gw *Gateway) checkBudgetWarning(msg channel.Message, lastCost float64) {
	budget := gw.cost.GetBudget()
	todayTotal, _, _ := gw.cost.GetToday()

	warnThreshold := budget.DailyLimitUSD * budget.WarnAtPercent
	if todayTotal >= warnThreshold && (todayTotal-lastCost) < warnThreshold {
		// 刚刚越过预警线
		remaining := budget.DailyLimitUSD - todayTotal
		gw.sendReply(msg, fmt.Sprintf(
			"⚠️ 预算预警！今日已用 $%.4f (%.0f%%)，剩余 $%.4f\n\n本地模型不计费，云端调用将在预算耗尽时暂停。",
			todayTotal, todayTotal/budget.DailyLimitUSD*100, remaining,
		))
	}
}

// maybeCompress 检查是否需要压缩上下文
func (gw *Gateway) maybeCompress(sessionID string) {
	history, err := gw.memory.GetHistory(sessionID, 100)
	if err != nil || len(history) < 40 {
		return // 不到 40 条不压缩
	}

	// 估算 token
	totalTokens := 0
	for _, e := range history {
		totalTokens += len([]rune(e.Content)) * 2 // 粗略估算
	}

	if totalTokens < 8000 {
		return
	}

	gw.logger.Info("触发上下文压缩", "session", sessionID, "messages", len(history), "est_tokens", totalTokens)

	// 用本地小模型做摘要（免费）
	summarizer := brain.NewLLMSummarizer(
		brain.NewOllamaProvider(gw.cfg.Models.Local.Endpoint),
		gw.cfg.Models.Local.Models.Small,
	)

	compressor := memory.NewCompressor(gw.memory, summarizer, gw.logger)
	if err := compressor.CheckAndCompress(context.Background(), sessionID); err != nil {
		gw.logger.Error("压缩失败", "error", err)
	}
}

// loadScenarios 加载配置中的场景包为定时任务
func (gw *Gateway) loadScenarios() {
	for name, sc := range gw.cfg.Scenarios {
		if !sc.Enabled || sc.Cron == "" {
			continue
		}
		// 场景包需要知道发送到哪个 chatID
		// P0: 用第一个发消息的用户的 chatID（启动后第一条消息时记录）
		gw.scheduler.AddJob(scheduler.Job{
			Name:    name,
			Cron:    sc.Cron,
			Prompt:  sc.Prompt,
			Enabled: true,
		})
	}
}

// handleScheduledJob 执行定时任务
func (gw *Gateway) handleScheduledJob(ctx context.Context, job scheduler.Job) error {
	gw.logger.Info("执行定时任务", "name", job.Name)

	// 构建消息
	messages := []brain.ChatMessage{
		{
			Role:    brain.RoleSystem,
			Content: "你是 LionClaw 🦁。以下是一个定时任务，请执行并返回结果。简洁实用，用中文。",
		},
		{
			Role:    brain.RoleUser,
			Content: job.Prompt,
		},
	}

	// 路由到模型
	provider, model, _, err := gw.router.Route(messages)
	if err != nil {
		return fmt.Errorf("模型路由失败: %w", err)
	}

	// 调用 LLM
	resp, err := provider.Chat(ctx, brain.ChatRequest{
		Messages: messages,
		Model:    model,
	})
	if err != nil {
		return fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 记录成本
	gw.cost.Record(brain.CostRecord{
		Model:        resp.Model,
		IsLocal:      provider.IsLocal(),
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
		TaskLabel:    "scenario:" + job.Name,
	})

	// 发送到 chatID
	reply := fmt.Sprintf("📋 [%s] 定时任务\n\n%s", job.Name, resp.Content)
	if job.ChatID != "" {
		for _, ch := range gw.channels {
			ch.Send(job.ChatID, reply, nil)
		}
	} else {
		gw.logger.Warn("定时任务无目标 chatID，跳过发送", "name", job.Name)
	}

	return nil
}

func (gw *Gateway) initChannels() error {
	// P0: 只支持 Telegram
	if gw.cfg.Channels.Telegram.Enabled {
		token, err := gw.vault.Get("TELEGRAM_BOT_TOKEN")
		if err != nil {
			return fmt.Errorf("Telegram Bot Token 未配置。运行: lionclaw vault set TELEGRAM_BOT_TOKEN <your-token>")
		}
		bot := channeltg.New(string(token), gw.logger)
		gw.channels = append(gw.channels, bot)
		gw.logger.Info("Telegram 渠道已初始化")
	}

	if len(gw.channels) == 0 {
		return fmt.Errorf("至少需要一个渠道。运行 `lionclaw setup` 配置")
	}

	return nil
}
