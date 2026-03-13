package github

import (
	"strings"
	"testing"
	"time"
)

func TestPRDescription(t *testing.T) {
	desc := BuildPRDescription("test-repo", "A test repo", "U12345", "We need this repo for the new microservice platform")
	if desc == "" {
		t.Fatal("expected non-empty PR description")
	}
}

func TestBranchName(t *testing.T) {
	name := BranchName("my-repo")
	if !strings.HasPrefix(name, "opsy/add-repo-my-repo-") {
		t.Errorf("got %q, want prefix %q", name, "opsy/add-repo-my-repo-")
	}
}

func TestBranchName_sanitizes(t *testing.T) {
	name := BranchName("My Repo With Spaces")
	if !strings.HasPrefix(name, "opsy/add-repo-my-repo-with-spaces-") {
		t.Errorf("got %q, want prefix %q", name, "opsy/add-repo-my-repo-with-spaces-")
	}
}

func TestBranchName_hasTimestamp(t *testing.T) {
	name := BranchName("repo")
	// should contain today's date
	today := time.Now().Format("20060102")
	if !strings.Contains(name, today) {
		t.Errorf("got %q, expected to contain date %q", name, today)
	}
}

func TestDeleteBranchName(t *testing.T) {
	name := DeleteBranchName("my-repo")
	if !strings.HasPrefix(name, "opsy/delete-repo-my-repo-") {
		t.Errorf("got %q, want prefix %q", name, "opsy/delete-repo-my-repo-")
	}
	today := time.Now().Format("20060102")
	if !strings.Contains(name, today) {
		t.Errorf("got %q, expected to contain date %q", name, today)
	}
}

func TestSettingsBranchName(t *testing.T) {
	name := SettingsBranchName("my-repo")
	if !strings.HasPrefix(name, "opsy/update-repo-my-repo-") {
		t.Errorf("got %q, want prefix %q", name, "opsy/update-repo-my-repo-")
	}
	today := time.Now().Format("20060102")
	if !strings.Contains(name, today) {
		t.Errorf("got %q, expected to contain date %q", name, today)
	}
}

func TestBuildSettingsPRDescription(t *testing.T) {
	desc := BuildSettingsPRDescription("test-repo", "John Doe", "Updating visibility")
	if !strings.Contains(desc, "Update repository settings: test-repo") {
		t.Error("missing repo name in description")
	}
	if !strings.Contains(desc, "Updating visibility") {
		t.Error("missing justification")
	}
	if !strings.Contains(desc, "John Doe") {
		t.Error("missing requester")
	}
}

func TestBuildDeletePRDescription(t *testing.T) {
	desc := BuildDeletePRDescription("test-repo", "John Doe", "No longer needed")
	if !strings.Contains(desc, "Remove repository: test-repo") {
		t.Error("missing repo name in description")
	}
	if !strings.Contains(desc, "No longer needed") {
		t.Error("missing justification")
	}
	if !strings.Contains(desc, "John Doe") {
		t.Error("missing requester")
	}
}
