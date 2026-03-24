package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Compressor 上下文压缩器
type Compressor struct {
	store      Store
	summarizer Summarizer
	logger     *slog.Logger
	maxTokens  int // 触发压缩的 token 阈值
}

// Summarizer 摘要生成接口（由 brain 层实现）
type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

// NewCompressor 创建压缩器
func NewCompressor(store Store, summarizer Summarizer, logger *slog.Logger) *Compressor {
	return &Compressor{
		store:      store,
		summarizer: summarizer,
		logger:     logger,
		maxTokens:  8000, // 超过 8K token 触发压缩
	}
}

// CheckAndCompress 检查并压缩指定会话
func (c *Compressor) CheckAndCompress(ctx context.Context, sessionID string) error {
	history, err := c.store.GetHistory(sessionID, 100)
	if err != nil {
		return err
	}

	// 估算总 token（粗略：1 个中文字 ≈ 2 token，1 个英文词 ≈ 1.5 token）
	totalTokens := 0
	for _, e := range history {
		totalTokens += estimateTokens(e.Content)
	}

	if totalTokens < c.maxTokens {
		return nil // 不需要压缩
	}

	c.logger.Info("触发上下文压缩",
		"session", sessionID,
		"messages", len(history),
		"est_tokens", totalTokens,
	)

	// 取前 2/3 的消息压缩为摘要
	splitIdx := len(history) * 2 / 3
	toCompress := history[:splitIdx]

	// 构建压缩文本
	var sb strings.Builder
	for _, e := range toCompress {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", e.Role, e.Content))
	}

	// 调用 LLM 摘要
	summary, err := c.summarizer.Summarize(ctx, sb.String())
	if err != nil {
		c.logger.Error("压缩失败，保留原文", "error", err)
		return nil // 不阻塞正常流程
	}

	c.logger.Info("压缩完成",
		"original_msgs", len(toCompress),
		"summary_len", len(summary),
	)

	// 存储压缩后的摘要（替换旧消息）
	// P0: 简单方案——将摘要作为 system 消息插入，保留最近 1/3 原文
	if err := c.store.SaveMessage(sessionID, Entry{
		Role:    "system",
		Content: fmt.Sprintf("[上下文摘要] %s", summary),
		Summary: summary,
	}); err != nil {
		return err
	}

	return nil
}

// estimateTokens 粗略估算 token 数
func estimateTokens(text string) int {
	// 简单估算：中文字数 × 2 + 英文词数 × 1.5
	runes := []rune(text)
	cjk := 0
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			cjk++
		}
	}
	ascii := len(runes) - cjk
	return cjk*2 + ascii*3/4
}
