package slack

import (
	"testing"

	goslack "github.com/slack-go/slack"
)

func buildRepoStep1Values(name, desc, visibility string) map[string]map[string]goslack.BlockAction {
	return map[string]map[string]goslack.BlockAction{
		BlockName:          {ElemName: {Value: name}},
		BlockDescription:   {ElemDescription: {Value: desc}},
		BlockVisibility:    {ElemVisibility: {SelectedOption: goslack.OptionBlockObject{Value: visibility}}},
		BlockJustification: {ElemJustification: {Value: "this is a test justification"}},
	}
}

func buildRepoStep2Values(branch string) map[string]map[string]goslack.BlockAction {
	return map[string]map[string]goslack.BlockAction{
		BlockDefBranch: {ElemDefBranch: {Value: branch}},
	}
}

func buildRepoStep3Values(reviews string) map[string]map[string]goslack.BlockAction {
	vals := map[string]map[string]goslack.BlockAction{}
	if reviews != "" {
		vals[BlockReviews] = map[string]goslack.BlockAction{
			ElemReviews: {Value: reviews},
		}
	}
	return vals
}

func TestValidateRepoStep1(t *testing.T) {
	tests := []struct {
		name      string
		repoName  string
		desc      string
		wantErr   map[string]bool
		wantClean bool
	}{
		{
			name:      "valid repo",
			repoName:  "my-new-repo",
			desc:      "A new repository",
			wantClean: true,
		},
		{
			name:     "empty name",
			repoName: "",
			desc:     "description",
			wantErr:  map[string]bool{BlockName: true},
		},
		{
			name:     "name with spaces",
			repoName: "my repo",
			desc:     "description",
			wantErr:  map[string]bool{BlockName: true},
		},
		{
			name:     "name starts with hyphen",
			repoName: "-my-repo",
			desc:     "description",
			wantErr:  map[string]bool{BlockName: true},
		},
		{
			name:     "name ends with dot",
			repoName: "my-repo.",
			desc:     "description",
			wantErr:  map[string]bool{BlockName: true},
		},
		{
			name:      "name with dots and underscores",
			repoName:  "my.repo_v2",
			desc:      "description",
			wantClean: true,
		},
		{
			name:     "empty description",
			repoName: "my-repo",
			desc:     "",
			wantErr:  map[string]bool{BlockDescription: true},
		},
		{
			name:     "both empty",
			repoName: "",
			desc:     "",
			wantErr:  map[string]bool{BlockName: true, BlockDescription: true},
		},
		{
			name:     "name too long",
			repoName: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			desc:     "description",
			wantErr:  map[string]bool{BlockName: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := buildRepoStep1Values(tt.repoName, tt.desc, "public")
			errs := validateRepoStep1(vals)

			if tt.wantClean {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %v", errs)
				}
				return
			}

			for blockID := range tt.wantErr {
				if _, ok := errs[blockID]; !ok {
					t.Errorf("expected error on %q, got none (errors: %v)", blockID, errs)
				}
			}
			for blockID := range errs {
				if !tt.wantErr[blockID] {
					t.Errorf("unexpected error on %q: %s", blockID, errs[blockID])
				}
			}
		})
	}
}

func TestValidateRepoStep2(t *testing.T) {
	tests := []struct {
		name      string
		branch    string
		wantErr   map[string]bool
		wantClean bool
	}{
		{name: "valid main", branch: "main", wantClean: true},
		{name: "valid develop", branch: "develop", wantClean: true},
		{name: "valid slash", branch: "feature/foo", wantClean: true},
		{name: "empty branch", branch: "", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "double dot", branch: "main..dev", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "space in name", branch: "main dev", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "starts with hyphen", branch: "-main", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "starts with dot", branch: ".hidden", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "ends with dot", branch: "main.", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "ends with slash", branch: "main/", wantErr: map[string]bool{BlockDefBranch: true}},
		{name: "tilde", branch: "main~1", wantErr: map[string]bool{BlockDefBranch: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := buildRepoStep2Values(tt.branch)
			errs := validateRepoStep2(vals)

			if tt.wantClean {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %v", errs)
				}
				return
			}

			for blockID := range tt.wantErr {
				if _, ok := errs[blockID]; !ok {
					t.Errorf("expected error on %q, got none (errors: %v)", blockID, errs)
				}
			}
		})
	}
}

func TestValidateRepoStep3(t *testing.T) {
	tests := []struct {
		name      string
		reviews   string
		wantErr   map[string]bool
		wantClean bool
	}{
		{name: "no reviews field", reviews: "", wantClean: true},
		{name: "valid 1", reviews: "1", wantClean: true},
		{name: "valid 5", reviews: "5", wantClean: true},
		{name: "zero", reviews: "0", wantErr: map[string]bool{BlockReviews: true}},
		{name: "six", reviews: "6", wantErr: map[string]bool{BlockReviews: true}},
		{name: "non-integer", reviews: "abc", wantErr: map[string]bool{BlockReviews: true}},
		{name: "negative", reviews: "-1", wantErr: map[string]bool{BlockReviews: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := buildRepoStep3Values(tt.reviews)
			errs := validateRepoStep3(vals)

			if tt.wantClean {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %v", errs)
				}
				return
			}

			for blockID := range tt.wantErr {
				if _, ok := errs[blockID]; !ok {
					t.Errorf("expected error on %q, got none (errors: %v)", blockID, errs)
				}
			}
		})
	}
}
