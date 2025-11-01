package config

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server   `toml:"server"`
	RAG      RAG      `toml:"rag"`
	Worker   Worker   `toml:"worker"` // 你在 CF 的 /search Worker
	DeepSeek DeepSeek `toml:"deepseek"`
	Redis    Redis    `toml:"redis"`
	DB       DB       `toml:"db"`
}

type Server struct {
	Port     string `toml:"port"`      // "8080"
	LogLevel string `toml:"log_level"` // "info" | "debug"
}

type RAG struct {
	TopK      int     `toml:"top_k"`     // 5
	Normalize bool    `toml:"normalize"` // true
	Threshold float64 `toml:"threshold"` // 0.35
}

type Worker struct {
	BaseURL string `toml:"base_url"` // https://xxx.workers.dev
	// Token   string `toml:"token"`   // 如需鉴权再加
}

type DeepSeek struct {
	APIKey       string `toml:"api_key"`
	ModelV3      string `toml:"model_v3"`      // deepseek-v3
	ModelDistill string `toml:"model_distill"` // deepseek-distill
	BaseUrl      string `toml:"base_url"`
}

type Redis struct {
	URL string `toml:"url"` // redis://localhost:6379/0
}

type DB struct {
	DSN string `toml:"dsn"` // postgres://user:pass@host:5432/db?sslmode=disable
}

// defaultConfig: 设置合理默认值（TOML 里没填就用它）
func defaultConfig() Config {
	return Config{
		Server: Server{
			Port:     "8080",
			LogLevel: "info",
		},
		RAG: RAG{
			TopK:      5,
			Normalize: true,
			Threshold: 0.35,
		},
	}
}

// Load 从 TOML 读取配置：优先顺序
// 1) 显式传入路径
// 2) 环境变量 CONFIG_PATH
// 3) 默认 "./config.toml"
// 注意：不存在文件时，会仅使用默认值（并做一次校验）
func Load(path ...string) (Config, error) {
	cfg := defaultConfig()

	// 选择路径
	p := "config.toml"
	if len(path) > 0 && strings.TrimSpace(path[0]) != "" {
		p = path[0]
	} else if v := os.Getenv("CONFIG_PATH"); v != "" {
		p = v
	}

	// 如果文件存在就 decode
	if _, err := os.Stat(p); err == nil {
		if _, err := toml.DecodeFile(p, &cfg); err != nil {
			return cfg, err
		}
	}

	// 基础校验与兜底
	if err := Validate(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Validate(c *Config) error {
	if c.RAG.TopK <= 0 {
		c.RAG.TopK = 5
	}
	if c.Worker.BaseURL != "" {
		if _, err := url.ParseRequestURI(c.Worker.BaseURL); err != nil {
			return errors.New("worker.base_url invalid")
		}
	}
	if c.DeepSeek.APIKey == "" {
		// 可以不强制；若必须，放开下面这行
		// return errors.New("deepseek.api_key required")
	}
	return nil
}
