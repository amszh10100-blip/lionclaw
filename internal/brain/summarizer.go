package brain

import (
	"context"
	"fmt"
)

// LLMSummarizer 用 LLM 做摘要（实现 memory.Summarizer 接口）
type LLMSummarizer struct {
	provider LLMProvider
	model    string
}

func NewLLMSummarizer(provider LLMProvider, model string) *LLMSummarizer {
	return &LLMSummarizer{provider: provider, model: model}
}

func (s *LLMSummarizer) Summarize(ctx context.Context, text string) (string, error) {
	if len(text) < 100 {
		return text, nil
	}

	// 截断过长的文本（避免超出模型上下文）
	if len(text) > 8000 {
		text = text[:8000]
	}

	req := ChatRequest{
		Messages: []ChatMessage{
			{
				Role:    RoleSystem,
				Content: "你是一个摘要助手。将以下对话历史压缩为简洁摘要，保留关键信息（人名、日期、决策、数字）。输出纯中文，不超过 500 字。",
			},
			{
				Role:    RoleUser,
				Content: fmt.Sprintf("请摘要以下对话：\n\n%s", text),
			},
		},
		Model: s.model,
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("摘要生成失败: %w", err)
	}

	return resp.Content, nil
}
