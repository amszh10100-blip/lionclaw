package brain

import (
	"context"
	"time"
)

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// ChatMessage LLM 对话消息
type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest LLM 调用请求
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatResponse LLM 调用响应
type ChatResponse struct {
	Content      string  `json:"content"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	LatencyMs    int64   `json:"latency_ms"`
	CostUSD      float64 `json:"cost_usd"`
}

// Complexity 任务复杂度
type Complexity int

const (
	ComplexityLow    Complexity = iota // 问候/导航 → 本地 8B
	ComplexityMedium                   // 对话/编程 → 本地 32B
	ComplexityHigh                     // 推理/创作 → 云端
)

// CostEstimate 成本预估
type CostEstimate struct {
	Model        string  `json:"model"`
	IsLocal      bool    `json:"is_local"`
	EstimatedUSD float64 `json:"estimated_usd"`
}

// CostRecord 成本记录
type CostRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Model        string    `json:"model"`
	IsLocal      bool      `json:"is_local"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	TaskLabel    string    `json:"task_label,omitempty"`
}

// Budget 预算设置
type Budget struct {
	DailyLimitUSD   float64 `json:"daily_limit_usd"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd"`
	WarnAtPercent   float64 `json:"warn_at_percent"`
}

// LLMProvider LLM 提供者接口
type LLMProvider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	IsLocal() bool
}

// Router 模型路由接口
type Router interface {
	Route(messages []ChatMessage) (provider LLMProvider, model string, est CostEstimate, err error)
}

// CostTracker 成本追踪接口
type CostTracker interface {
	Record(record CostRecord) error
	GetToday() (total float64, records []CostRecord, err error)
	GetMonth() (total float64, records []CostRecord, err error)
	CheckBudget(estimated float64) (allowed bool, remaining float64, err error)
	GetBudget() Budget
	SetBudget(b Budget) error
}
