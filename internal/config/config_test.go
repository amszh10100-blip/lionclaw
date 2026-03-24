package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Security.Bind != "127.0.0.1" {
		t.Errorf("default bind = %s, want 127.0.0.1", cfg.Security.Bind)
	}
	if cfg.Security.Port != 18790 {
		t.Errorf("default port = %d, want 18790", cfg.Security.Port)
	}
	if cfg.Cost.DailyLimitUSD != 5.0 {
		t.Errorf("daily limit = %f, want 5.0", cfg.Cost.DailyLimitUSD)
	}
	if !cfg.Models.Local.Enabled {
		t.Error("local models should be enabled by default")
	}
	if cfg.Channels.Telegram.Enabled {
		t.Error("telegram should be disabled by default")
	}
	if len(cfg.Routing.PrivacyKeywords) == 0 {
		t.Error("should have default privacy keywords")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// 临时 HOME
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Cost.DailyLimitUSD = 10.0

	// 创建配置目录
	os.MkdirAll(filepath.Join(tmpDir, ".goldlion"), 0700)

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !loaded.Channels.Telegram.Enabled {
		t.Error("loaded telegram should be enabled")
	}
	if loaded.Cost.DailyLimitUSD != 10.0 {
		t.Errorf("loaded daily = %f, want 10", loaded.Cost.DailyLimitUSD)
	}
}

func TestLoadMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err := Load()
	if err == nil {
		t.Error("Load should fail when config missing")
	}
}
