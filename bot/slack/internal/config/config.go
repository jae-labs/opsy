package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SlackAppToken string
	SlackBotToken string
	GitHubToken   string
	GitHubOwner   string
	GitHubRepo    string
}

func Load() (*Config, error) {
	// best-effort load; missing .env is fine in production
	_ = godotenv.Load()

	cfg := &Config{
		SlackAppToken: os.Getenv("SLACK_APP_TOKEN"),
		SlackBotToken: os.Getenv("SLACK_BOT_TOKEN"),
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		GitHubOwner:   os.Getenv("GITHUB_OWNER"),
		GitHubRepo:    os.Getenv("GITHUB_REPO"),
	}

	if cfg.SlackAppToken == "" || cfg.SlackBotToken == "" || cfg.GitHubToken == "" || cfg.GitHubOwner == "" || cfg.GitHubRepo == "" {
		return nil, fmt.Errorf("missing required env vars: SLACK_APP_TOKEN, SLACK_BOT_TOKEN, GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO")
	}
	return cfg, nil
}
