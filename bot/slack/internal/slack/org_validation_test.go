package slack

import (
	"testing"

	goslack "github.com/slack-go/slack"
)

func orgSettingsValues(name, email, blog, desc string) map[string]map[string]goslack.BlockAction {
	return map[string]map[string]goslack.BlockAction{
		BlockOrgName:    {ElemOrgName: {Value: name}},
		BlockOrgBilling: {ElemOrgBilling: {Value: email}},
		BlockOrgBlog:    {ElemOrgBlog: {Value: blog}},
		BlockOrgDesc:    {ElemOrgDesc: {Value: desc}},
	}
}

func TestValidateOrgSettings(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]map[string]goslack.BlockAction
		wantErr   string // block ID expected in errors, or "" for valid
	}{
		{
			name:   "valid config",
			values: orgSettingsValues("My Org", "a@b.com", "https://example.com", "A description"),
		},
		{
			name:    "empty name",
			values:  orgSettingsValues("", "a@b.com", "", "A description"),
			wantErr: BlockOrgName,
		},
		{
			name:    "invalid email no at",
			values:  orgSettingsValues("Org", "invalid", "", "desc"),
			wantErr: BlockOrgBilling,
		},
		{
			name:    "empty email",
			values:  orgSettingsValues("Org", "", "", "desc"),
			wantErr: BlockOrgBilling,
		},
		{
			name:    "invalid blog url",
			values:  orgSettingsValues("Org", "a@b.com", "ftp://bad", "desc"),
			wantErr: BlockOrgBlog,
		},
		{
			name:   "empty blog is valid",
			values: orgSettingsValues("Org", "a@b.com", "", "desc"),
		},
		{
			name:    "empty description",
			values:  orgSettingsValues("Org", "a@b.com", "", ""),
			wantErr: BlockOrgDesc,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateOrgSettings(tc.values)
			if tc.wantErr == "" {
				if len(errs) > 0 {
					t.Errorf("expected no errors, got %v", errs)
				}
			} else {
				if _, ok := errs[tc.wantErr]; !ok {
					t.Errorf("expected error for %s, got %v", tc.wantErr, errs)
				}
			}
		})
	}
}
