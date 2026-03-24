package channel

import (
	"context"
	"time"
)

// Message 是跨渠道的统一消息格式
type Message struct {
	ID        string            `json:"id"`
	ChatID    string            `json:"chat_id"`
	UserID    string            `json:"user_id"`
	Text      string            `json:"text"`
	Timestamp time.Time         `json:"timestamp"`
	ReplyTo   string            `json:"reply_to,omitempty"`
	Files     []Attachment      `json:"files,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// Attachment 消息附件
type Attachment struct {
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Data     []byte `json:"-"`
	URL      string `json:"url,omitempty"`
}

// SendOptions 发送选项
type SendOptions struct {
	ReplyTo string     `json:"reply_to,omitempty"`
	Buttons [][]Button `json:"buttons,omitempty"`
	Silent  bool       `json:"silent,omitempty"`
}

// Button 内联按钮
type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// Channel 是所有消息渠道的统一接口
// P0 只实现 Telegram，但接口设计为可扩展
type Channel interface {
	// Name 返回渠道标识（"telegram", "discord" 等）
	Name() string

	// Start 启动渠道监听，阻塞直到 ctx 取消
	Start(ctx context.Context) error

	// Stop 优雅关闭
	Stop() error

	// Send 发送消息到指定聊天
	Send(chatID string, text string, opts *SendOptions) error

	// OnMessage 注册消息回调处理函数
	OnMessage(handler func(msg Message))
}
