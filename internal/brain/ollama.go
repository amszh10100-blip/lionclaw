package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider 本地 Ollama 模型
type OllamaProvider struct {
	endpoint string
	client   *http.Client
}

// NewOllamaProvider 创建 Ollama 提供者
func NewOllamaProvider(endpoint string) *OllamaProvider {
	return &OllamaProvider{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (o *OllamaProvider) Name() string  { return "ollama" }
func (o *OllamaProvider) IsLocal() bool { return true }

// ollamaChatReq Ollama /api/chat 请求格式
type ollamaChatReq struct {
	Model    string             `json:"model"`
	Messages []ollamaChatMsg    `json:"messages"`
	Stream   bool               `json:"stream"`
	Options  map[string]any     `json:"options,omitempty"`
}

type ollamaChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Message struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking,omitempty"` // Qwen3 thinking 模式
	} `json:"message"`
	TotalDuration   int64 `json:"total_duration"` // nanoseconds
	PromptEvalCount int   `json:"prompt_eval_count"`
	EvalCount       int   `json:"eval_count"`
}

func (o *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()

	// 转换消息格式
	msgs := make([]ollamaChatMsg, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ollamaChatMsg{Role: string(m.Role), Content: m.Content}
	}

	body := ollamaChatReq{
		Model:    req.Model,
		Messages: msgs,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.endpoint+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama 调用失败: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("Ollama 返回 %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp ollamaChatResp
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &ChatResponse{
		Content:      resp.Message.Content,
		Model:        req.Model,
		InputTokens:  resp.PromptEvalCount,
		OutputTokens: resp.EvalCount,
		LatencyMs:    time.Since(start).Milliseconds(),
		CostUSD:      0, // 本地模型零成本
	}, nil
}

// Ping 检查 Ollama 是否运行
func (o *OllamaProvider) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", o.endpoint+"/api/tags", nil)
	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama 不可达: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama 返回 %d", resp.StatusCode)
	}
	return nil
}
