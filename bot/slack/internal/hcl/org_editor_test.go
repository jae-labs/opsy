package hcl

import (
	"os"
	"testing"
)

func loadOrgTestdata(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("testdata/locals_org.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	return src
}

func TestOrgExtractAllFields(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, err := ExtractOrgSettings(src)
	if err != nil {
		t.Fatalf("ExtractOrgSettings: %v", err)
	}

	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Name", cfg.Name, "JAE Labs"},
		{"BillingEmail", cfg.BillingEmail, "luiz@justanother.engineer"},
		{"Blog", cfg.Blog, "https://justanother.engineer"},
		{"Description", cfg.Description, "Just Another Engineer playing with code."},
		{"Location", cfg.Location, "Ireland"},
		{"MembersCanCreateRepos", cfg.MembersCanCreateRepos, false},
		{"DefaultRepoPermission", cfg.DefaultRepoPermission, "read"},
		{"WebCommitSignoffRequired", cfg.WebCommitSignoffRequired, false},
		{"DependabotAlerts", cfg.DependabotAlerts, true},
		{"DependabotSecurityUpdates", cfg.DependabotSecurityUpdates, true},
		{"DependencyGraph", cfg.DependencyGraph, true},
	}
	for _, tc := range checks {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestOrgUpdateStringField(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, _ := ExtractOrgSettings(src)
	cfg.Description = "Updated description"

	out, err := UpdateOrgSettings(src, cfg)
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}

	updated, err := ExtractOrgSettings(out)
	if err != nil {
		t.Fatalf("ExtractOrgSettings after update: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", updated.Description, "Updated description")
	}
	// other fields should be unchanged
	if updated.Name != cfg.Name {
		t.Errorf("Name changed unexpectedly: %q", updated.Name)
	}
}

func TestOrgUpdateBoolField(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, _ := ExtractOrgSettings(src)
	cfg.MembersCanCreateRepos = true

	out, err := UpdateOrgSettings(src, cfg)
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}

	updated, err := ExtractOrgSettings(out)
	if err != nil {
		t.Fatalf("ExtractOrgSettings after update: %v", err)
	}
	if !updated.MembersCanCreateRepos {
		t.Error("MembersCanCreateRepos should be true")
	}
}

func TestOrgUpdateMultipleFields(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, _ := ExtractOrgSettings(src)
	cfg.Description = "Multi-update test"
	cfg.Location = "Portugal"
	cfg.DependabotAlerts = false
	cfg.WebCommitSignoffRequired = true

	out, err := UpdateOrgSettings(src, cfg)
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}

	updated, err := ExtractOrgSettings(out)
	if err != nil {
		t.Fatalf("ExtractOrgSettings after update: %v", err)
	}
	if updated.Description != "Multi-update test" {
		t.Errorf("Description = %q", updated.Description)
	}
	if updated.Location != "Portugal" {
		t.Errorf("Location = %q", updated.Location)
	}
	if updated.DependabotAlerts {
		t.Error("DependabotAlerts should be false")
	}
	if !updated.WebCommitSignoffRequired {
		t.Error("WebCommitSignoffRequired should be true")
	}
}

func TestOrgUpdateNoOp(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, _ := ExtractOrgSettings(src)

	out, err := UpdateOrgSettings(src, cfg)
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}
	if string(out) != string(src) {
		t.Error("expected no changes when config is unchanged")
	}
}

func TestOrgRoundTrip(t *testing.T) {
	src := loadOrgTestdata(t)
	cfg, _ := ExtractOrgSettings(src)
	cfg.Blog = "https://example.com"
	cfg.DefaultRepoPermission = "admin"

	out, err := UpdateOrgSettings(src, cfg)
	if err != nil {
		t.Fatalf("UpdateOrgSettings: %v", err)
	}

	roundTripped, err := ExtractOrgSettings(out)
	if err != nil {
		t.Fatalf("ExtractOrgSettings round-trip: %v", err)
	}

	if roundTripped != cfg {
		t.Errorf("round-trip mismatch:\ngot  %+v\nwant %+v", roundTripped, cfg)
	}
}


