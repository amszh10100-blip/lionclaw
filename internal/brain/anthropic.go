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

// AnthropicProvider Claude API
type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

const anthropicAPI = "https://api.anthropic.com/v1/messages"

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *AnthropicProvider) Name() string  { return "anthropic" }
func (a *AnthropicProvider) IsLocal() bool { return false }

type anthropicReq struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// 价格表（每百万 token 美元）
var anthropicPricing = map[string][2]float64{
	"claude-opus-4-6":            {15.0, 75.0},
	"claude-sonnet-4-6":          {3.0, 15.0},
	"claude-3-7-sonnet-20250219": {3.0, 15.0},
}

func (a *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()

	// 提取 system prompt
	var system string
	var msgs []anthropicMsg
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
			continue
		}
		msgs = append(msgs, anthropicMsg{Role: string(m.Role), Content: m.Content})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := anthropicReq{
		Model:     req.Model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", anthropicAPI, bytes.NewReader(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic 调用失败: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("Anthropic 返回 %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp anthropicResp
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	// 计算成本
	costUSD := calculateAnthropicCost(req.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)

	return &ChatResponse{
		Content:      content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		LatencyMs:    time.Since(start).Milliseconds(),
		CostUSD:      costUSD,
	}, nil
}

func calculateAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := anthropicPricing[model]
	if !ok {
		// 未知模型，用 Sonnet 价格估算
		pricing = [2]float64{3.0, 15.0}
	}
	inputCost := float64(inputTokens) / 1_000_000 * pricing[0]
	outputCost := float64(outputTokens) / 1_000_000 * pricing[1]
	return inputCost + outputCost
}
