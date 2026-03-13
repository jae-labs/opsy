package slack

import (
	"testing"

	"github.com/jae-labs/opsy/internal/conversation"
)

func TestWelcomeBlocks(t *testing.T) {
	blocks := WelcomeBlocks()
	if len(blocks) == 0 {
		t.Fatal("expected non-empty blocks")
	}
}

func TestResourceBlocks_known(t *testing.T) {
	blocks := ResourceBlocks("github")
	if len(blocks) == 0 {
		t.Fatal("expected non-empty blocks")
	}
}

func TestResourceBlocks_unknown(t *testing.T) {
	blocks := ResourceBlocks("unknown")
	if len(blocks) == 0 {
		t.Fatal("expected coming soon blocks")
	}
}

func TestRepoStep1Modal(t *testing.T) {
	modal := RepoStep1Modal()
	if modal.Title == nil {
		t.Fatal("expected modal title")
	}
	if modal.CallbackID != CallbackRepoStep1 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackRepoStep1)
	}
}

func TestRepoStep2Modal(t *testing.T) {
	teams := []string{"Maintainers", "Developers"}
	modal := RepoStep2Modal(teams)
	if modal.CallbackID != CallbackRepoStep2 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackRepoStep2)
	}
}

func TestRepoStep3Modal(t *testing.T) {
	modal := RepoStep3Modal()
	if modal.CallbackID != CallbackRepoStep3 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackRepoStep3)
	}
}

func TestActionBlocks(t *testing.T) {
	blocks := ActionBlocks("repo")
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestActionBlocks_unknown(t *testing.T) {
	blocks := ActionBlocks("unknown")
	if len(blocks) == 0 {
		t.Fatal("expected coming soon blocks")
	}
}

func TestDeleteRepoModal(t *testing.T) {
	modal := DeleteRepoModal([]string{"repo-a", "repo-b"})
	if modal.CallbackID != CallbackDeleteRepo {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackDeleteRepo)
	}
	if len(modal.Blocks.BlockSet) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestDeleteConfirmationBlocks(t *testing.T) {
	blocks := DeleteConfirmationBlocks("my-repo", "No longer needed for the project")
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestConfirmationBlocks(t *testing.T) {
	blocks := ConfirmationBlocks("test-repo", "desc", "public", []string{"go"}, map[string]string{"Maintainers": "admin"}, "main", true, false, false, false, false, 0, false, false, false, false, false, "", "This repo is needed for the new service")
	if len(blocks) == 0 {
		t.Fatal("expected non-empty blocks")
	}
}

func TestSelectRepoModal(t *testing.T) {
	modal := SelectRepoModal([]string{"repo-a", "repo-b"})
	if modal.CallbackID != CallbackSelectRepo {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackSelectRepo)
	}
	if len(modal.Blocks.BlockSet) != 1 {
		t.Errorf("expected 1 block, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestSettingsStep1Modal(t *testing.T) {
	cfg := conversation.RepoConfig{
		Name:        "my-repo",
		Description: "My repo",
		Visibility:  "public",
	}
	modal := SettingsStep1Modal(cfg)
	if modal.CallbackID != CallbackSettingsStep1 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackSettingsStep1)
	}
	// 4 blocks: context + description + visibility + justification
	if len(modal.Blocks.BlockSet) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestSettingsStep2Modal(t *testing.T) {
	cfg := conversation.RepoConfig{
		Topics:        []string{"go", "cli"},
		TeamAccess:    map[string]string{"Maintainers": "admin"},
		DefaultBranch: "main",
	}
	modal := SettingsStep2Modal(cfg, []string{"Maintainers", "Developers"})
	if modal.CallbackID != CallbackSettingsStep2 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackSettingsStep2)
	}
	// 4 blocks: topics + 2 team role dropdowns + default branch
	if len(modal.Blocks.BlockSet) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestSettingsStep3Modal(t *testing.T) {
	cfg := conversation.RepoConfig{
		EnableBranchProtection: true,
		RequiredReviews:        2,
		DismissStaleReviews:    true,
		AllowAutoMerge:         true,
	}
	modal := SettingsStep3Modal(cfg)
	if modal.CallbackID != CallbackSettingsStep3 {
		t.Errorf("got callback=%q, want %q", modal.CallbackID, CallbackSettingsStep3)
	}
}

func TestSettingsConfirmationBlocks(t *testing.T) {
	oldCfg := conversation.RepoConfig{
		Description: "Old desc",
		Visibility:  "public",
	}
	newCfg := conversation.RepoConfig{
		Description: "New desc",
		Visibility:  "private",
	}
	blocks := SettingsConfirmationBlocks("my-repo", oldCfg, newCfg, "Need to update settings")
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}
