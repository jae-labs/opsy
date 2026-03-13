package hcl

import (
	"os"
	"strings"
	"testing"

	"github.com/jae-labs/opsy/internal/conversation"
)

func TestAddRepo_basic(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	repo := conversation.RepoConfig{
		Name:          "new-project",
		Description:   "A shiny new project",
		Visibility:    "public",
		HasIssues:     true,
		DefaultBranch: "main",
		Topics:        []string{"golang", "api"},
		TeamAccess:    map[string]string{"Maintainers": "admin"},
	}

	result, err := AddRepo(src, repo)
	if err != nil {
		t.Fatalf("AddRepo: %v", err)
	}

	output := string(result)

	if !strings.Contains(output, `"new-project"`) {
		t.Error("output missing new-project repo name")
	}
	if !strings.Contains(output, `A shiny new project`) {
		t.Error("output missing description")
	}
	if !strings.Contains(output, `"golang"`) {
		t.Error("output missing topics")
	}
	if !strings.Contains(output, `"terraform"`) {
		t.Error("output missing existing terraform repo")
	}
	if !strings.Contains(output, `"catv"`) {
		t.Error("output missing existing catv repo")
	}

	// verify valid HCL
	_, err = Parse(result)
	if err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestAddRepo_duplicate(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	repo := conversation.RepoConfig{
		Name:          "terraform",
		Description:   "duplicate",
		Visibility:    "public",
		HasIssues:     true,
		DefaultBranch: "main",
		TeamAccess:    map[string]string{"Maintainers": "admin"},
	}

	_, err = AddRepo(src, repo)
	if err == nil {
		t.Fatal("expected error for duplicate repo name")
	}
}

func TestAddRepo_withBranchProtection(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	repo := conversation.RepoConfig{
		Name:                          "protected-repo",
		Description:                   "A protected repo",
		Visibility:                    "private",
		HasIssues:                     true,
		DefaultBranch:                 "main",
		TeamAccess:                    map[string]string{"Maintainers": "admin"},
		EnableBranchProtection:        true,
		RequiredReviews:               1,
		DismissStaleReviews:           true,
		RequireLinearHistory:          true,
		RequireConversationResolution: true,
		AllowAutoMerge:                true,
		DeleteBranchOnMerge:           true,
	}

	result, err := AddRepo(src, repo)
	if err != nil {
		t.Fatalf("AddRepo: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "branch_protection") {
		t.Error("output missing branch_protection block")
	}
	if !strings.Contains(output, "required_reviews") {
		t.Error("output missing required_reviews")
	}
	if !strings.Contains(output, "allow_auto_merge") {
		t.Error("output missing allow_auto_merge")
	}

	_, err = Parse(result)
	if err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestRemoveRepo(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	result, err := RemoveRepo(src, "catv")
	if err != nil {
		t.Fatalf("RemoveRepo: %v", err)
	}

	output := string(result)
	if strings.Contains(output, `"catv"`) {
		t.Error("output still contains catv repo")
	}
	if !strings.Contains(output, `"terraform"`) {
		t.Error("output missing existing terraform repo")
	}
	if !strings.Contains(output, `"scripts"`) {
		t.Error("output missing existing scripts repo")
	}

	_, err = Parse(result)
	if err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestRemoveRepo_notFound(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	_, err = RemoveRepo(src, "nonexistent-repo")
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestExistingRepoNames(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	names, err := ExistingRepoNames(src)
	if err != nil {
		t.Fatalf("ExistingRepoNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one repo name")
	}

	found := false
	for _, n := range names {
		if n == "terraform" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'terraform' in repo names")
	}
}

func TestExtractRepoConfig(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, err := ExtractRepoConfig(src, "catv")
	if err != nil {
		t.Fatalf("ExtractRepoConfig: %v", err)
	}

	if cfg.Name != "catv" {
		t.Errorf("name = %q, want %q", cfg.Name, "catv")
	}
	if cfg.Description != "Transform notes into flashcards with local AI." {
		t.Errorf("description = %q", cfg.Description)
	}
	if cfg.Visibility != "public" {
		t.Errorf("visibility = %q, want public", cfg.Visibility)
	}
	if !cfg.HasIssues {
		t.Error("expected has_issues = true")
	}
	if cfg.HasWiki {
		t.Error("expected has_wiki = false (field absent)")
	}
	if !cfg.AllowAutoMerge {
		t.Error("expected allow_auto_merge = true")
	}
	if !cfg.AllowUpdateBranch {
		t.Error("expected allow_update_branch = true")
	}
	if !cfg.DeleteBranchOnMerge {
		t.Error("expected delete_branch_on_merge = true")
	}
	if cfg.DefaultBranch != "main" {
		t.Errorf("default_branch = %q, want main", cfg.DefaultBranch)
	}
	if len(cfg.Topics) != 5 {
		t.Errorf("topics count = %d, want 5", len(cfg.Topics))
	}
	if cfg.TeamAccess["Maintainers"] != "admin" {
		t.Errorf("team_access = %v", cfg.TeamAccess)
	}
	if cfg.EnableBranchProtection {
		t.Error("expected branch_protection disabled (null)")
	}
}

func TestExtractRepoConfig_withProtection(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, err := ExtractRepoConfig(src, "terraform")
	if err != nil {
		t.Fatalf("ExtractRepoConfig: %v", err)
	}

	if !cfg.EnableBranchProtection {
		t.Fatal("expected branch_protection enabled")
	}
	if cfg.RequiredReviews != 1 {
		t.Errorf("required_reviews = %d, want 1", cfg.RequiredReviews)
	}
	if !cfg.DismissStaleReviews {
		t.Error("expected dismiss_stale_reviews = true")
	}
	if !cfg.RequireLinearHistory {
		t.Error("expected require_linear_history = true")
	}
	if !cfg.RequireConversationResolution {
		t.Error("expected require_conversation_resolution = true")
	}
}

func TestUpdateRepo_changeDescription(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, _ := ExtractRepoConfig(src, "catv")
	cfg.Description = "Updated description for catv"

	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}

	if !strings.Contains(string(result), "Updated description for catv") {
		t.Error("output missing updated description")
	}
	// verify existing repos still present
	if !strings.Contains(string(result), `"terraform"`) {
		t.Error("output missing terraform repo")
	}
	// verify valid HCL
	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestUpdateRepo_addOptionalField(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, _ := ExtractRepoConfig(src, "catv")
	cfg.HasWiki = true

	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}

	if !strings.Contains(string(result), "has_wiki") {
		t.Error("output missing has_wiki field")
	}
	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestUpdateRepo_removeOptionalField(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, _ := ExtractRepoConfig(src, "catv")
	cfg.AllowAutoMerge = false

	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	// round-trip: extract config back and verify the field is now false
	extracted, err := ExtractRepoConfig(result, "catv")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted.AllowAutoMerge {
		t.Error("expected allow_auto_merge to be false after removal")
	}
	// verify other optional bools were preserved
	if !extracted.AllowUpdateBranch {
		t.Error("expected allow_update_branch to remain true")
	}
	if !extracted.DeleteBranchOnMerge {
		t.Error("expected delete_branch_on_merge to remain true")
	}
}

func TestUpdateRepo_toggleBranchProtection(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	// enable protection on catv (was null)
	cfg, _ := ExtractRepoConfig(src, "catv")
	cfg.EnableBranchProtection = true
	cfg.RequiredReviews = 2
	cfg.DismissStaleReviews = true
	cfg.RequireLinearHistory = false
	cfg.RequireConversationResolution = true

	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo enable: %v", err)
	}

	if !strings.Contains(string(result), "required_reviews") {
		t.Error("output missing required_reviews after enabling protection")
	}
	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	// disable protection on terraform (was enabled)
	cfg2, _ := ExtractRepoConfig(src, "terraform")
	cfg2.EnableBranchProtection = false

	result2, err := UpdateRepo(src, "terraform", cfg2)
	if err != nil {
		t.Fatalf("UpdateRepo disable: %v", err)
	}

	// verify the terraform block now has null protection
	// extract the config back and verify
	cfg3, err := ExtractRepoConfig(result2, "terraform")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if cfg3.EnableBranchProtection {
		t.Error("expected branch_protection to be disabled")
	}
}

func TestUpdateRepo_noChanges(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, _ := ExtractRepoConfig(src, "catv")
	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo: %v", err)
	}

	if string(result) != string(src) {
		t.Error("expected no changes, but output differs from input")
	}
}

func TestExtractTeamNames(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_members.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	teams, err := ExtractTeamNames(src)
	if err != nil {
		t.Fatalf("ExtractTeamNames: %v", err)
	}

	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}
	if teams[0] != "Maintainers" {
		t.Errorf("got %q, want Maintainers", teams[0])
	}
}

func TestExtractRepoConfig_newFields(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	cfg, err := ExtractRepoConfig(src, "community-hub")
	if err != nil {
		t.Fatalf("ExtractRepoConfig: %v", err)
	}

	if !cfg.HasDiscussions {
		t.Error("expected has_discussions = true")
	}
	if !cfg.HasProjects {
		t.Error("expected has_projects = true")
	}
	if cfg.HomepageURL != "https://community.justanother.engineer" {
		t.Errorf("homepage_url = %q, want https://community.justanother.engineer", cfg.HomepageURL)
	}
}

func TestAddRepo_withNewFields(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	repo := conversation.RepoConfig{
		Name:           "new-with-fields",
		Description:    "Repo with new fields",
		Visibility:     "public",
		HasIssues:      true,
		HasDiscussions: true,
		HasProjects:    true,
		HomepageURL:    "https://example.com",
		DefaultBranch:  "main",
		TeamAccess:     map[string]string{"Maintainers": "admin"},
	}

	result, err := AddRepo(src, repo)
	if err != nil {
		t.Fatalf("AddRepo: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "has_discussions") {
		t.Error("output missing has_discussions")
	}
	if !strings.Contains(output, "has_projects") {
		t.Error("output missing has_projects")
	}
	if !strings.Contains(output, "https://example.com") {
		t.Error("output missing homepage_url")
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestUpdateRepo_toggleNewFields(t *testing.T) {
	src, err := os.ReadFile("testdata/locals_repos.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	// enable new fields on catv (which doesn't have them)
	cfg, _ := ExtractRepoConfig(src, "catv")
	cfg.HasDiscussions = true
	cfg.HasProjects = true
	cfg.HomepageURL = "https://catv.dev"

	result, err := UpdateRepo(src, "catv", cfg)
	if err != nil {
		t.Fatalf("UpdateRepo add fields: %v", err)
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	extracted, err := ExtractRepoConfig(result, "catv")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if !extracted.HasDiscussions {
		t.Error("expected has_discussions = true after adding")
	}
	if !extracted.HasProjects {
		t.Error("expected has_projects = true after adding")
	}
	if extracted.HomepageURL != "https://catv.dev" {
		t.Errorf("homepage_url = %q, want https://catv.dev", extracted.HomepageURL)
	}

	// now disable them on community-hub (which has them)
	cfg2, _ := ExtractRepoConfig(src, "community-hub")
	cfg2.HasDiscussions = false
	cfg2.HasProjects = false
	cfg2.HomepageURL = ""

	result2, err := UpdateRepo(src, "community-hub", cfg2)
	if err != nil {
		t.Fatalf("UpdateRepo remove fields: %v", err)
	}

	if _, err := Parse(result2); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	extracted2, err := ExtractRepoConfig(result2, "community-hub")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted2.HasDiscussions {
		t.Error("expected has_discussions = false after removal")
	}
	if extracted2.HasProjects {
		t.Error("expected has_projects = false after removal")
	}
	if extracted2.HomepageURL != "" {
		t.Errorf("expected homepage_url empty after removal, got %q", extracted2.HomepageURL)
	}
}
