package brain

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/lionclaw/lionclaw/internal/config"
)

// DefaultRouter 智能模型路由器
type DefaultRouter struct {
	cfg        *config.Config
	cost       CostTracker
	logger     *slog.Logger
	localSmall LLMProvider
	localLarge LLMProvider
	cloud      LLMProvider
	cloudModel string
}

func NewRouter(cfg *config.Config, cost CostTracker, logger *slog.Logger) (*DefaultRouter, error) {
	r := &DefaultRouter{
		cfg:    cfg,
		cost:   cost,
		logger: logger,
	}

	// 初始化本地模型
	if cfg.Models.Local.Enabled {
		ollama := NewOllamaProvider(cfg.Models.Local.Endpoint)
		r.localSmall = ollama
		r.localLarge = ollama
		logger.Info("本地模型已配置",
			"small", cfg.Models.Local.Models.Small,
			"large", cfg.Models.Local.Models.Large,
		)
	}

	// 初始化云端模型
	// 注意：API Key 从 Vault 获取，这里先用占位
	// TODO: 从 Vault 获取 API Key
	if cfg.Models.Cloud.Anthropic.Enabled {
		r.cloudModel = cfg.Models.Cloud.Anthropic.Model
		logger.Info("云端模型已配置", "model", r.cloudModel)
	}

	return r, nil
}

// SetCloudProvider 设置云端提供者（Vault 解密后调用）
func (r *DefaultRouter) SetCloudProvider(provider LLMProvider, model string) {
	r.cloud = provider
	r.cloudModel = model
}

func (r *DefaultRouter) Route(messages []ChatMessage) (LLMProvider, string, CostEstimate, error) {
	if len(messages) == 0 {
		return nil, "", CostEstimate{}, fmt.Errorf("消息列表为空")
	}

	lastMsg := messages[len(messages)-1].Content

	// 1. 隐私检查（最高优先级）
	if r.containsPrivacyKeywords(lastMsg) {
		r.logger.Info("隐私关键词命中，强制本地模型")
		return r.pickLocal(true)
	}

	// 2. 复杂度判断
	complexity := r.assessComplexity(lastMsg, messages)

	switch complexity {
	case ComplexityLow:
		return r.pickLocal(false)
	case ComplexityMedium:
		return r.pickLocal(true)
	case ComplexityHigh:
		// 优先云端，预算不够降级本地
		if r.cloud != nil {
			allowed, _, _ := r.cost.CheckBudget(0.05)
			if allowed {
				return r.cloud, r.cloudModel, CostEstimate{
					Model:        r.cloudModel,
					IsLocal:      false,
					EstimatedUSD: 0.05,
				}, nil
			}
			r.logger.Warn("预算不足，降级到本地模型")
		}
		return r.pickLocal(true)
	}

	return r.pickLocal(false)
}

func (r *DefaultRouter) pickLocal(large bool) (LLMProvider, string, CostEstimate, error) {
	if large && r.localLarge != nil {
		model := r.cfg.Models.Local.Models.Large
		r.logger.Debug("选择本地大模型", "model", model)
		return r.localLarge, model, CostEstimate{Model: model, IsLocal: true}, nil
	}
	if r.localSmall != nil {
		model := r.cfg.Models.Local.Models.Small
		r.logger.Debug("选择本地小模型", "model", model)
		return r.localSmall, model, CostEstimate{Model: model, IsLocal: true}, nil
	}
	// 本地都没有，试云端
	if r.cloud != nil {
		return r.cloud, r.cloudModel, CostEstimate{Model: r.cloudModel, IsLocal: false, EstimatedUSD: 0.05}, nil
	}
	return nil, "", CostEstimate{}, fmt.Errorf("无可用模型：本地和云端均未配置")
}

func (r *DefaultRouter) containsPrivacyKeywords(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range r.cfg.Routing.PrivacyKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func (r *DefaultRouter) assessComplexity(text string, history []ChatMessage) Complexity {
	score := 0
	runeCount := utf8.RuneCountInString(text)

	// 长度
	if runeCount > 200 {
		score += 2
	}
	if runeCount > 500 {
		score += 2
	}

	// 中复杂度关键词
	medKeywords := []string{
		"怎么", "如何", "请问", "帮我", "解释", "翻译",
		"how", "what", "explain", "translate", "help",
	}
	for _, kw := range medKeywords {
		if strings.Contains(strings.ToLower(text), kw) {
			score += 1
			break
		}
	}

	// 高复杂度关键词（中英文）
	highKeywords := []string{
		"为什么", "分析", "比较", "设计", "架构", "策略", "深度",
		"why", "analyze", "compare", "design", "implement", "architect",
		"代码", "code", "debug", "优化", "重构", "算法",
		"写一个", "写一段", "开发", "研究", "报告", "总结全文", "实现",
	}
	highHits := 0
	for _, kw := range highKeywords {
		if strings.Contains(strings.ToLower(text), kw) {
			highHits++
		}
	}
	if highHits >= 2 {
		score += 3 // 多个高复杂度关键词 = 一定是高复杂度
	} else if highHits == 1 {
		score += 2
	}

	// 简单消息检测
	if isGreeting(text) {
		return ComplexityLow
	}
	if runeCount < 10 {
		return ComplexityLow
	}

	if score >= 3 {
		return ComplexityHigh
	}
	if score >= 1 {
		return ComplexityMedium
	}
	return ComplexityLow
}

func isGreeting(text string) bool {
	greetings := []string{
		"你好", "hi", "hello", "hey", "嗨", "早上好", "晚上好",
		"morning", "good morning", "谢谢", "thanks", "ok", "好的",
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, g := range greetings {
		if lower == g {
			return true
		}
	}
	return false
}
