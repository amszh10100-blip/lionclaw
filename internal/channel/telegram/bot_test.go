package telegram

import (
	"testing"
	"log/slog"
)

func TestNewBot(t *testing.T) {
	logger := slog.Default()
	bot := New("dummy_token", logger)
	if bot == nil {
		t.Fatal("Bot should not be nil")
	}
}
