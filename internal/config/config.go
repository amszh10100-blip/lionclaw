package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 是 LionClaw 的顶级配置结构
type Config struct {
	Channels  ChannelsConfig            `yaml:"channels"`
	Models    ModelsConfig              `yaml:"models"`
	Routing   RoutingConfig             `yaml:"routing"`
	Cost      CostConfig                `yaml:"cost"`
	Security  SecurityConfig            `yaml:"security"`
	Scenarios map[string]ScenarioConfig `yaml:"scenarios"`
}

type ChannelsConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
}

type TelegramConfig struct {
	Enabled bool `yaml:"enabled"`
	// Token 存在 Vault 中，不在配置文件
}

type ModelsConfig struct {
	Local LocalModelsConfig `yaml:"local"`
	Cloud CloudModelsConfig `yaml:"cloud"`
}

type LocalModelsConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
	Models   struct {
		Small string `yaml:"small"`
		Large string `yaml:"large"`
	} `yaml:"models"`
}

type CloudModelsConfig struct {
	Anthropic struct {
		Enabled bool   `yaml:"enabled"`
		Model   string `yaml:"model"`
	} `yaml:"anthropic"`
	OpenAI struct {
		Enabled bool   `yaml:"enabled"`
		Model   string `yaml:"model"`
	} `yaml:"openai"`
}

type RoutingConfig struct {
	PrivacyKeywords []string `yaml:"privacy_keywords"`
}

type CostConfig struct {
	DailyLimitUSD   float64 `yaml:"daily_limit_usd"`
	MonthlyLimitUSD float64 `yaml:"monthly_limit_usd"`
	WarnAtPercent   float64 `yaml:"warn_at_percent"`
}

type SecurityConfig struct {
	Bind string `yaml:"bind"`
	Port int    `yaml:"port"`
	WebUI struct { 
		User string `yaml:"user"` 
		Pass string `yaml:"pass"` 
	} `yaml:"webui"` 

}

type ScenarioConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Cron          string `yaml:"cron,omitempty"`
	MinutesBefore int    `yaml:"minutes_before,omitempty"`
	Prompt        string `yaml:"prompt"`
}

// DefaultConfig 返回安全的默认配置
func DefaultConfig() *Config {
	return &Config{
		Channels: ChannelsConfig{
			Telegram: TelegramConfig{Enabled: false},
		},
		Models: ModelsConfig{
			Local: LocalModelsConfig{
				Enabled:  true,
				Endpoint: func() string { if e := os.Getenv("OLLAMA_HOST"); e != "" { return e }; return "http://127.0.0.1:11434" }(),
				Models: struct {
					Small string `yaml:"small"`
					Large string `yaml:"large"`
				}{
					Small: "qwen3:8b",
					Large: "qwen3:30b",
				},
			},
		},
		Routing: RoutingConfig{
			PrivacyKeywords: []string{
				"密码", "password", "银行", "身份证",
				"token", "secret", "api_key", "private_key",
			},
		},
		Cost: CostConfig{
			DailyLimitUSD:   5.0,
			MonthlyLimitUSD: 50.0,
			WarnAtPercent:   0.8,
		},
		Security: SecurityConfig{
			Bind: "127.0.0.1",
			Port: 18790,
		},
	}
}

// ConfigDir 返回配置目录路径
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lionclaw")
}

// ConfigPath 返回配置文件路径
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// DataDir 返回数据目录路径
func DataDir() string {
	return filepath.Join(ConfigDir(), "data")
}

// Load 从文件加载配置
func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("配置文件不存在: %s\n请先运行 `lionclaw setup`", path)
		}
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return cfg, nil
}

// Save 保存配置到文件
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	return nil
}
