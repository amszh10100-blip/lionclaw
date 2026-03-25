package gateway

import (
	"testing"

	"github.com/lionclaw/lionclaw/internal/channel"
	"github.com/lionclaw/lionclaw/internal/config"
	"github.com/lionclaw/lionclaw/internal/memory"
)

func TestGateway_Scenarios(t *testing.T) {
	cfg := config.DefaultConfig()
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	chatID := "test-chat"

	// 默认应该是 assistant
	msgs := gw.buildMessages(chatID, []memory.Entry{}, "hello")
	if len(msgs) != 2 || msgs[0].Content != builtinScenarios[0].SystemPrompt {
		t.Errorf("Expected default system prompt %q, got %q", builtinScenarios[0].SystemPrompt, msgs[0].Content)
	}

	// 切换到 coder
	gw.cmdSetScenario(channel.Message{ChatID: chatID, Text: "/scenario coder"}, "coder")

	msgs = gw.buildMessages(chatID, []memory.Entry{}, "hello")
	if msgs[0].Content != builtinScenarios[2].SystemPrompt {
		t.Errorf("Expected coder system prompt %q, got %q", builtinScenarios[2].SystemPrompt, msgs[0].Content)
	}

	// 切换到未知场景
	gw.cmdSetScenario(channel.Message{ChatID: chatID, Text: "/scenario unknown"}, "unknown")
	
	// 应该是 fallback 之前的状态 coder 还是没变？
	msgs = gw.buildMessages(chatID, []memory.Entry{}, "hello")
	if msgs[0].Content != builtinScenarios[2].SystemPrompt {
		t.Errorf("Expected unchanged system prompt, got %q", msgs[0].Content)
	}
}
