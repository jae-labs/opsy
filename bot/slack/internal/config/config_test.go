package config

import (
	"testing"
)

func TestLoad_missingRequired(t *testing.T) {
	for _, key := range []string{"SLACK_APP_TOKEN", "SLACK_BOT_TOKEN", "GITHUB_TOKEN", "GITHUB_OWNER", "GITHUB_REPO"} {
		t.Setenv(key, "")
	}
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing env vars")
	}
}

func TestLoad_valid(t *testing.T) {
	t.Setenv("SLACK_APP_TOKEN", "xapp-test")
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	t.Setenv("GITHUB_TOKEN", "ghp_test")
	t.Setenv("GITHUB_OWNER", "test-org")
	t.Setenv("GITHUB_REPO", "test-repo")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SlackAppToken != "xapp-test" {
		t.Errorf("got SlackAppToken=%q, want xapp-test", cfg.SlackAppToken)
	}
	if cfg.GitHubOwner != "test-org" {
		t.Errorf("got GitHubOwner=%q, want test-org", cfg.GitHubOwner)
	}
}
