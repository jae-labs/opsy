package hcl

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jae-labs/opsy/internal/conversation"
)

// ExtractOrgSettings reads the org_settings block from HCL source.
func ExtractOrgSettings(src []byte) (conversation.OrgConfig, error) {
	obj, err := findOrgSettingsObject(src)
	if err != nil {
		return conversation.OrgConfig{}, err
	}

	var cfg conversation.OrgConfig
	for _, item := range obj.Items {
		fieldName, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		switch fieldName {
		case "name":
			cfg.Name, _ = exprToString(item.ValueExpr)
		case "billing_email":
			cfg.BillingEmail, _ = exprToString(item.ValueExpr)
		case "blog":
			cfg.Blog, _ = exprToString(item.ValueExpr)
		case "description":
			cfg.Description, _ = exprToString(item.ValueExpr)
		case "location":
			cfg.Location, _ = exprToString(item.ValueExpr)
		case "members_can_create_repositories":
			cfg.MembersCanCreateRepos, _ = exprToBool(item.ValueExpr)
		case "default_repository_permission":
			cfg.DefaultRepoPermission, _ = exprToString(item.ValueExpr)
		case "web_commit_signoff_required":
			cfg.WebCommitSignoffRequired, _ = exprToBool(item.ValueExpr)
		case "dependabot_alerts_enabled_for_new_repositories":
			cfg.DependabotAlerts, _ = exprToBool(item.ValueExpr)
		case "dependabot_security_updates_enabled_for_new_repositories":
			cfg.DependabotSecurityUpdates, _ = exprToBool(item.ValueExpr)
		case "dependency_graph_enabled_for_new_repositories":
			cfg.DependencyGraph, _ = exprToBool(item.ValueExpr)
		}
	}

	return cfg, nil
}

// UpdateOrgSettings performs in-place field-level edits on the org_settings block.
func UpdateOrgSettings(src []byte, cfg conversation.OrgConfig) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	obj, err := findOrgSettingsObject(src)
	if err != nil {
		return nil, err
	}

	fieldMap := make(map[string]hclsyntax.ObjectConsItem)
	for _, item := range obj.Items {
		k, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		fieldMap[k] = item
	}

	var edits []textEdit

	replaceValue := func(item hclsyntax.ObjectConsItem, newText string) {
		edits = append(edits, textEdit{
			start: item.ValueExpr.Range().Start.Byte,
			end:   item.ValueExpr.Range().End.Byte,
			text:  newText,
		})
	}

	// string fields
	stringFields := []struct {
		hclKey string
		value  string
	}{
		{"name", cfg.Name},
		{"billing_email", cfg.BillingEmail},
		{"blog", cfg.Blog},
		{"description", cfg.Description},
		{"location", cfg.Location},
		{"default_repository_permission", cfg.DefaultRepoPermission},
	}
	for _, sf := range stringFields {
		if item, ok := fieldMap[sf.hclKey]; ok {
			old, _ := exprToString(item.ValueExpr)
			if old != sf.value {
				replaceValue(item, fmt.Sprintf("%q", sf.value))
			}
		}
	}

	// bool fields
	boolFields := []struct {
		hclKey string
		value  bool
	}{
		{"members_can_create_repositories", cfg.MembersCanCreateRepos},
		{"web_commit_signoff_required", cfg.WebCommitSignoffRequired},
		{"dependabot_alerts_enabled_for_new_repositories", cfg.DependabotAlerts},
		{"dependabot_security_updates_enabled_for_new_repositories", cfg.DependabotSecurityUpdates},
		{"dependency_graph_enabled_for_new_repositories", cfg.DependencyGraph},
	}
	for _, bf := range boolFields {
		if item, ok := fieldMap[bf.hclKey]; ok {
			old, _ := exprToBool(item.ValueExpr)
			if old != bf.value {
				replaceValue(item, boolStr(bf.value))
			}
		}
	}

	if len(edits) == 0 {
		return src, nil
	}

	// sort descending to apply in reverse byte order
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].start > edits[j].start
	})

	out := make([]byte, len(src))
	copy(out, src)
	for _, e := range edits {
		var buf bytes.Buffer
		buf.Write(out[:e.start])
		buf.WriteString(e.text)
		buf.Write(out[e.end:])
		out = buf.Bytes()
	}

	if _, err := Parse(out); err != nil {
		return nil, fmt.Errorf("modified HCL is invalid: %w", err)
	}
	return out, nil
}

// findOrgSettingsObject locates the org_settings attribute in the locals block.
func findOrgSettingsObject(src []byte) (*hclsyntax.ObjectConsExpr, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return nil, err
	}

	attr, ok := localsBody.Attributes["org_settings"]
	if !ok {
		return nil, fmt.Errorf("org_settings attribute not found in locals block")
	}

	obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("org_settings is not an object expression")
	}
	return obj, nil
}
