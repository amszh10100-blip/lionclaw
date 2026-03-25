package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/lionclaw/lionclaw/internal/channel"
	"github.com/lionclaw/lionclaw/internal/config"
)

type mockChannel struct {
	lastSend string
}

func (m *mockChannel) Name() string { return "mock" }
func (m *mockChannel) Start(ctx context.Context) error { return nil }
func (m *mockChannel) Stop() error { return nil }
func (m *mockChannel) OnMessage(handler func(channel.Message)) {}
func (m *mockChannel) Send(chatID string, text string, opts *channel.SendOptions) error {
	m.lastSend = text
	return nil
}

func TestCmdShare(t *testing.T) {
	cfg := config.DefaultConfig()
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	
	mc := &mockChannel{}
	gw.channels = append(gw.channels, mc)

	gw.cmdShare(channel.Message{ChatID: "test", Text: "/share"})
	
	if !strings.Contains(mc.lastSend, "LionClaw AI Agent") {
		t.Errorf("Expected share card to contain 'LionClaw AI Agent', got %s", mc.lastSend)
	}
	if !strings.Contains(mc.lastSend, "今日对话") {
		t.Errorf("Expected share card to contain '今日对话', got %s", mc.lastSend)
	}
}
