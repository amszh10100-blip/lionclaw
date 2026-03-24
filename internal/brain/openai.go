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

// OpenAIProvider OpenAI API (兼容 GPT-5.x 系列)
type OpenAIProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

const defaultOpenAIEndpoint = "https://api.openai.com/v1/chat/completions"

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:   apiKey,
		endpoint: defaultOpenAIEndpoint,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (o *OpenAIProvider) Name() string  { return "openai" }
func (o *OpenAIProvider) IsLocal() bool { return false }

type openaiReq struct {
	Model    string       `json:"model"`
	Messages []openaiMsg  `json:"messages"`
}

type openaiMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

var openaiPricing = map[string][2]float64{
	"gpt-5.1":       {2.0, 8.0},
	"gpt-5.1-codex": {2.0, 8.0},
	"gpt-5.4":       {5.0, 20.0},
	"gpt-4o":        {2.5, 10.0},
}

func (o *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()

	msgs := make([]openaiMsg, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openaiMsg{Role: string(m.Role), Content: m.Content}
	}

	body := openaiReq{Model: req.Model, Messages: msgs}
	jsonBody, _ := json.Marshal(body)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", o.endpoint, bytes.NewReader(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	httpResp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI 调用失败: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("OpenAI 返回 %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp openaiResp
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	costUSD := calculateOpenAICost(req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &ChatResponse{
		Content:      content,
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		LatencyMs:    time.Since(start).Milliseconds(),
		CostUSD:      costUSD,
	}, nil
}

func calculateOpenAICost(model string, input, output int) float64 {
	pricing, ok := openaiPricing[model]
	if !ok {
		pricing = [2]float64{2.0, 8.0}
	}
	return float64(input)/1_000_000*pricing[0] + float64(output)/1_000_000*pricing[1]
}
