package main

import (
	"log/slog"
	"os"

	"github.com/jae-labs/opsy/internal/config"
	ghclient "github.com/jae-labs/opsy/internal/github"
	slackhandler "github.com/jae-labs/opsy/internal/slack"
	slacklib "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	api := slacklib.New(
		cfg.SlackBotToken,
		slacklib.OptionAppLevelToken(cfg.SlackAppToken),
	)

	sm := socketmode.New(api)

	gh := ghclient.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)

	handler := slackhandler.NewHandler(api, sm, gh, logger)

	slog.Info("opsy starting in socket mode")
	handler.Run()
}
