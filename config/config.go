package config

import "github.com/rbrick/clanker/env"

type TelegramConfig struct {
	BotKey string `env:"TELEGRAM_BOT_KEY" json:"bot_key"`
}

type AIConfig struct {
	Provider string `env:"LLM_PROVIDER;default:openai" json:"provider"`
	APIKey   string `env:"LLM_API_KEY" json:"api_key"`
	Model    string `env:"LLM_MODEL;default:gpt-5.4-mini" json:"model"`
}

type DatabaseConfig struct {
	URI      string `env:"DB_URI;default:clanker.db" json:"uri"`
	Provider string `env:"DB_PROVIDER;default:sqlite" json:"provider"`
}

type Config struct {
	Telegram TelegramConfig `json:"telegram"`
	AI       AIConfig       `json:"ai"`
	Database DatabaseConfig `json:"database"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg, false)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
