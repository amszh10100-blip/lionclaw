package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/goldlion/goldlion/internal/channel"
)

// Bot Telegram Bot 实现
type Bot struct {
	token   string
	client  *http.Client
	handler func(channel.Message)
	logger  *slog.Logger
	stopCh  chan struct{}
	offset  int64
}

// New 创建 Telegram Bot
func New(token string, logger *slog.Logger) *Bot {
	return &Bot{
		token:  token,
		client: &http.Client{Timeout: 60 * time.Second},
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (b *Bot) Name() string { return "telegram" }

// Start 启动长轮询
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("Telegram Bot 启动轮询")

	// 验证 token
	me, err := b.getMe()
	if err != nil {
		return fmt.Errorf("Telegram Bot Token 无效: %w", err)
	}
	b.logger.Info("Telegram Bot 已连接", "username", me.Username, "id", me.ID)

	// 长轮询循环
	go b.pollLoop(ctx)

	// 发送启动通知（如果有之前的聊天）
	b.logger.Info("Telegram Bot 轮询已启动，等待消息")

	return nil
}

func (b *Bot) Stop() error {
	close(b.stopCh)
	return nil
}

func (b *Bot) OnMessage(handler func(channel.Message)) {
	b.handler = handler
}

// Send 发送消息
func (b *Bot) Send(chatID string, text string, opts *channel.SendOptions) error {
	// 不设 parse_mode，避免 Markdown 语法错误导致发送失败
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	if opts != nil && opts.ReplyTo != "" {
		msgID, _ := strconv.ParseInt(opts.ReplyTo, 10, 64)
		if msgID > 0 {
			payload["reply_parameters"] = map[string]any{
				"message_id": msgID,
			}
		}
	}

	// 内联键盘按钮
	if opts != nil && len(opts.Buttons) > 0 {
		var rows [][]map[string]string
		for _, row := range opts.Buttons {
			var btns []map[string]string
			for _, btn := range row {
				btns = append(btns, map[string]string{
					"text":          btn.Text,
					"callback_data": btn.CallbackData,
				})
			}
			rows = append(rows, btns)
		}
		payload["reply_markup"] = map[string]any{
			"inline_keyboard": rows,
		}
	}

	_, err := b.apiCall("sendMessage", payload)
	return err
}

// --- Telegram API 类型 ---

type tgUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	IsBot    bool   `json:"is_bot"`
}

type tgMessage struct {
	MessageID int64   `json:"message_id"`
	From      *tgUser `json:"from"`
	Chat      tgChat  `json:"chat"`
	Text      string  `json:"text"`
	Date      int64   `json:"date"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Desc   string          `json:"description"`
}

// --- 内部方法 ---

func (b *Bot) pollLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		default:
		}

		updates, err := b.getUpdates(b.offset, 30)
		if err != nil {
			b.logger.Error("轮询失败", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(updates) > 0 {
			b.logger.Info("收到 Telegram updates", "count", len(updates))
		}

		for _, update := range updates {
			b.offset = update.UpdateID + 1

			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			msg := channel.Message{
				ID:        strconv.FormatInt(update.Message.MessageID, 10),
				ChatID:    strconv.FormatInt(update.Message.Chat.ID, 10),
				UserID:    strconv.FormatInt(update.Message.From.ID, 10),
				Text:      update.Message.Text,
				Timestamp: time.Unix(update.Message.Date, 0),
				Meta: map[string]string{
					"chat_type": update.Message.Chat.Type,
				},
			}

			if update.Message.From != nil {
				msg.Meta["username"] = update.Message.From.Username
			}

			if b.handler != nil {
				go b.handler(msg)
			}
		}
	}
}

func (b *Bot) getUpdates(offset int64, timeout int) ([]tgUpdate, error) {
	params := map[string]any{
		"offset":          offset,
		"timeout":         timeout,
		"allowed_updates": []string{"message"},
	}

	data, err := b.apiCall("getUpdates", params)
	if err != nil {
		return nil, err
	}

	var updates []tgUpdate
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("解析 updates 失败: %w", err)
	}

	return updates, nil
}

func (b *Bot) getMe() (*tgUser, error) {
	data, err := b.apiCall("getMe", nil)
	if err != nil {
		return nil, err
	}

	var user tgUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (b *Bot) apiCall(method string, payload map[string]any) (json.RawMessage, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var tgResp tgResponse
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if !tgResp.OK {
		return nil, fmt.Errorf("Telegram API 错误: %s", tgResp.Desc)
	}

	return tgResp.Result, nil
}
