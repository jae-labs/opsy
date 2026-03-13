package hcl

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jae-labs/opsy/internal/conversation"
	"github.com/zclconf/go-cty/cty"
)

// Parse validates HCL source and returns the hclwrite AST.
func Parse(src []byte) (*hclwrite.File, error) {
	f, diags := hclwrite.ParseConfig(src, "locals.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse HCL: %s", diags.Error())
	}
	return f, nil
}

// AddRepo inserts a new repository entry into the repos map within the locals block.
// It validates the source HCL, checks for duplicate repo names, inserts the new
// entry via text manipulation, and re-validates the result.
func AddRepo(src []byte, repo conversation.RepoConfig) ([]byte, error) {
	// validate input HCL
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	// check for duplicate repo names using hclsyntax for reading values
	existing, err := ExistingRepoNames(src)
	if err != nil {
		return nil, fmt.Errorf("read existing repos: %w", err)
	}
	for _, name := range existing {
		if name == repo.Name {
			return nil, fmt.Errorf("repository %q already exists", repo.Name)
		}
	}

	// generate the new repo HCL block
	entry, err := renderRepoEntry(repo)
	if err != nil {
		return nil, fmt.Errorf("render repo entry: %w", err)
	}

	// find insertion point (closing brace of repos map)
	offset, err := findReposClosingBrace(src)
	if err != nil {
		return nil, fmt.Errorf("find repos closing brace: %w", err)
	}

	// insert new entry before the closing brace of repos
	var result bytes.Buffer
	result.Write(src[:offset])
	result.WriteString(entry)
	result.Write(src[offset:])

	out := result.Bytes()

	// re-validate the modified output
	if _, err := Parse(out); err != nil {
		return nil, fmt.Errorf("modified HCL is invalid: %w", err)
	}

	return out, nil
}

// RemoveRepo removes a repository entry from the repos map within the locals block.
func RemoveRepo(src []byte, repoName string) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	start, end, err := findRepoRange(src, repoName)
	if err != nil {
		return nil, err
	}

	// strip trailing whitespace/newlines after the entry
	for end < len(src) && (src[end] == '\n' || src[end] == '\r') {
		end++
	}

	var result bytes.Buffer
	result.Write(src[:start])
	result.Write(src[end:])
	out := result.Bytes()

	if _, err := Parse(out); err != nil {
		return nil, fmt.Errorf("modified HCL is invalid: %w", err)
	}
	return out, nil
}

// findRepoRange returns the start and end byte offsets for a repo entry in the repos map.
func findRepoRange(src []byte, repoName string) (int, int, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return 0, 0, err
	}

	reposAttr, ok := localsBody.Attributes["repos"]
	if !ok {
		return 0, 0, fmt.Errorf("repos attribute not found in locals block")
	}

	objExpr, ok := reposAttr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return 0, 0, fmt.Errorf("repos is not an object expression")
	}

	for _, item := range objExpr.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		if name != repoName {
			continue
		}

		// hclsyntax ranges are 1-indexed line/col; use byte offsets from source range
		keyStart := item.KeyExpr.Range().Start.Byte
		valEnd := item.ValueExpr.Range().End.Byte

		// find the start of the line containing the key
		start := keyStart
		for start > 0 && src[start-1] != '\n' {
			start--
		}

		// find the end of the line containing the value end (closing brace)
		end := valEnd
		for end < len(src) && src[end] != '\n' {
			end++
		}
		if end < len(src) {
			end++ // include the newline
		}

		return start, end, nil
	}

	return 0, 0, fmt.Errorf("repository %q not found", repoName)
}

// ExtractTeamNames reads team names from the teams map in locals.
func ExtractTeamNames(src []byte) ([]string, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return nil, err
	}

	teamsAttr, ok := localsBody.Attributes["teams"]
	if !ok {
		return nil, fmt.Errorf("teams attribute not found in locals block")
	}

	objExpr, ok := teamsAttr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("teams is not an object expression")
	}

	var names []string
	for _, item := range objExpr.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			return nil, fmt.Errorf("read team name: %w", err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// localsBlockBody parses the HCL source and returns the body of the "locals" block.
func localsBlockBody(src []byte) (*hclsyntax.Body, error) {
	file, diags := hclsyntax.ParseConfig(src, "locals.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse HCL: %s", diags.Error())
	}

	body := file.Body.(*hclsyntax.Body)

	for _, block := range body.Blocks {
		if block.Type == "locals" {
			return block.Body, nil
		}
	}
	return nil, fmt.Errorf("locals block not found")
}

// ExistingRepoNames extracts repo names from the repos map using hclsyntax.
func ExistingRepoNames(src []byte) ([]string, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return nil, err
	}

	reposAttr, ok := localsBody.Attributes["repos"]
	if !ok {
		return nil, fmt.Errorf("repos attribute not found in locals block")
	}

	objExpr, ok := reposAttr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("repos is not an object expression")
	}

	var names []string
	for _, item := range objExpr.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			return nil, fmt.Errorf("read repo name: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

// exprToString extracts a string value from an HCL expression.
// handles ObjectConsKeyExpr, TemplateExpr (quoted strings), and TemplateWrapExpr.
func exprToString(expr hclsyntax.Expression) (string, error) {
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsKeyExpr:
		// unwrap the key expression and recurse
		return exprToString(e.Wrapped)
	case *hclsyntax.TemplateExpr:
		if e.IsStringLiteral() {
			val, diags := e.Value(nil)
			if diags.HasErrors() {
				return "", fmt.Errorf("evaluate string: %s", diags.Error())
			}
			return val.AsString(), nil
		}
		return "", fmt.Errorf("non-literal template expression")
	case *hclsyntax.TemplateWrapExpr:
		val, diags := e.Value(nil)
		if diags.HasErrors() {
			return "", fmt.Errorf("evaluate string: %s", diags.Error())
		}
		if val.Type() == cty.String {
			return val.AsString(), nil
		}
		return "", fmt.Errorf("template wrap expression is not a string")
	case *hclsyntax.ScopeTraversalExpr:
		// bare identifiers used as object keys (e.g. description, visibility)
		if len(e.Traversal) == 1 {
			return e.Traversal.RootName(), nil
		}
		return "", fmt.Errorf("complex traversal expression")
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			return e.Val.AsString(), nil
		}
		return "", fmt.Errorf("literal value is not a string")
	default:
		return "", fmt.Errorf("unsupported expression type %T", expr)
	}
}

// exprToBool extracts a bool value from an HCL LiteralValueExpr.
func exprToBool(expr hclsyntax.Expression) (bool, error) {
	lit, ok := expr.(*hclsyntax.LiteralValueExpr)
	if !ok {
		return false, fmt.Errorf("expected LiteralValueExpr, got %T", expr)
	}
	if lit.Val.Type() != cty.Bool {
		return false, fmt.Errorf("expected bool, got %s", lit.Val.Type().FriendlyName())
	}
	return lit.Val.True(), nil
}

// exprToInt extracts an int value from an HCL LiteralValueExpr.
func exprToInt(expr hclsyntax.Expression) (int, error) {
	lit, ok := expr.(*hclsyntax.LiteralValueExpr)
	if !ok {
		return 0, fmt.Errorf("expected LiteralValueExpr, got %T", expr)
	}
	if lit.Val.Type() != cty.Number {
		return 0, fmt.Errorf("expected number, got %s", lit.Val.Type().FriendlyName())
	}
	bf := lit.Val.AsBigFloat()
	i, acc := bf.Int(nil)
	if acc != big.Exact {
		return 0, fmt.Errorf("number is not an integer: %s", bf.String())
	}
	return int(i.Int64()), nil
}

// findRepoObject returns the inner ObjectConsExpr for a named repo entry.
func findRepoObject(src []byte, repoName string) (*hclsyntax.ObjectConsExpr, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return nil, err
	}

	reposAttr, ok := localsBody.Attributes["repos"]
	if !ok {
		return nil, fmt.Errorf("repos attribute not found in locals block")
	}

	objExpr, ok := reposAttr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("repos is not an object expression")
	}

	for _, item := range objExpr.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		if name != repoName {
			continue
		}
		inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			return nil, fmt.Errorf("repo %q value is not an object", repoName)
		}
		return inner, nil
	}
	return nil, fmt.Errorf("repository %q not found", repoName)
}

// ExtractRepoConfig reads a repo's configuration from HCL source.
// Unknown fields (e.g. environments) are ignored.
func ExtractRepoConfig(src []byte, repoName string) (conversation.RepoConfig, error) {
	obj, err := findRepoObject(src, repoName)
	if err != nil {
		return conversation.RepoConfig{}, err
	}

	cfg := conversation.RepoConfig{Name: repoName, HasIssues: true}

	for _, item := range obj.Items {
		key, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		switch key {
		case "description":
			cfg.Description, _ = exprToString(item.ValueExpr)
		case "visibility":
			cfg.Visibility, _ = exprToString(item.ValueExpr)
		case "default_branch":
			cfg.DefaultBranch, _ = exprToString(item.ValueExpr)
		case "has_issues":
			cfg.HasIssues, _ = exprToBool(item.ValueExpr)
		case "has_wiki":
			cfg.HasWiki, _ = exprToBool(item.ValueExpr)
		case "has_discussions":
			cfg.HasDiscussions, _ = exprToBool(item.ValueExpr)
		case "has_projects":
			cfg.HasProjects, _ = exprToBool(item.ValueExpr)
		case "homepage_url":
			cfg.HomepageURL, _ = exprToString(item.ValueExpr)
		case "allow_auto_merge":
			cfg.AllowAutoMerge, _ = exprToBool(item.ValueExpr)
		case "allow_update_branch":
			cfg.AllowUpdateBranch, _ = exprToBool(item.ValueExpr)
		case "delete_branch_on_merge":
			cfg.DeleteBranchOnMerge, _ = exprToBool(item.ValueExpr)
		case "topics":
			tuple, ok := item.ValueExpr.(*hclsyntax.TupleConsExpr)
			if ok {
				for _, elem := range tuple.Exprs {
					if s, err := exprToString(elem); err == nil {
						cfg.Topics = append(cfg.Topics, s)
					}
				}
			}
		case "team_access":
			inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
			if ok {
				cfg.TeamAccess = make(map[string]string)
				for _, ta := range inner.Items {
					k, _ := exprToString(ta.KeyExpr)
					v, _ := exprToString(ta.ValueExpr)
					if k != "" && v != "" {
						cfg.TeamAccess[k] = v
					}
				}
			}
		case "branch_protection":
			// null means disabled
			if lit, ok := item.ValueExpr.(*hclsyntax.LiteralValueExpr); ok && lit.Val.IsNull() {
				cfg.EnableBranchProtection = false
				continue
			}
			inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
			if !ok {
				continue
			}
			cfg.EnableBranchProtection = true
			for _, bp := range inner.Items {
				bpKey, _ := exprToString(bp.KeyExpr)
				switch bpKey {
				case "required_reviews":
					cfg.RequiredReviews, _ = exprToInt(bp.ValueExpr)
				case "dismiss_stale_reviews":
					cfg.DismissStaleReviews, _ = exprToBool(bp.ValueExpr)
				case "require_linear_history":
					cfg.RequireLinearHistory, _ = exprToBool(bp.ValueExpr)
				case "require_conversation_resolution":
					cfg.RequireConversationResolution, _ = exprToBool(bp.ValueExpr)
				}
			}
		}
	}

	return cfg, nil
}

// textEdit represents a byte-range replacement in the source.
type textEdit struct {
	start int
	end   int
	text  string
}

// removeFieldEdit creates an edit that removes an entire field line (inclusive of newline).
func removeFieldEdit(src []byte, item hclsyntax.ObjectConsItem) textEdit {
	start := item.KeyExpr.Range().Start.Byte
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	end := item.ValueExpr.Range().End.Byte
	for end < len(src) && src[end] != '\n' {
		end++
	}
	if end < len(src) {
		end++ // include newline
	}
	return textEdit{start: start, end: end, text: ""}
}

// addFieldInsertPoint returns the byte offset just before the repo's closing '}'.
func addFieldInsertPoint(src []byte, repoObj *hclsyntax.ObjectConsExpr) int {
	end := repoObj.SrcRange.End.Byte
	// walk backwards past the closing '}'
	pos := end - 1
	for pos > 0 && src[pos] != '}' {
		pos--
	}
	// now pos is at '}', find start of that line
	lineStart := pos
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	return lineStart
}

func formatTopics(topics []string) string {
	parts := make([]string, len(topics))
	for i, t := range topics {
		parts[i] = fmt.Sprintf("%q", t)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatTeamAccess(teams map[string]string) string {
	var keys []string
	for k := range teams {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%q = %q", k, teams[k]))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// extractTeamAccessFromItem reads team access from an existing HCL ObjectConsExpr.
func extractTeamAccessFromItem(item hclsyntax.ObjectConsItem) map[string]string {
	inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil
	}
	teams := make(map[string]string)
	for _, ta := range inner.Items {
		k, _ := exprToString(ta.KeyExpr)
		v, _ := exprToString(ta.ValueExpr)
		if k != "" && v != "" {
			teams[k] = v
		}
	}
	return teams
}

func teamMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func formatBranchProtection(cfg conversation.RepoConfig) string {
	return fmt.Sprintf(`{
        required_reviews                = %d
        dismiss_stale_reviews           = %s
        require_linear_history          = %s
        require_conversation_resolution = %s
      }`, cfg.RequiredReviews, boolStr(cfg.DismissStaleReviews),
		boolStr(cfg.RequireLinearHistory), boolStr(cfg.RequireConversationResolution))
}

// UpdateRepo performs in-place field-level edits on an existing repo entry,
// preserving unknown fields like environments.
func UpdateRepo(src []byte, repoName string, cfg conversation.RepoConfig) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	repoObj, err := findRepoObject(src, repoName)
	if err != nil {
		return nil, err
	}

	// build field map from existing items
	fieldMap := make(map[string]hclsyntax.ObjectConsItem)
	for _, item := range repoObj.Items {
		key, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		fieldMap[key] = item
	}

	var edits []textEdit

	// helper: replace value range with new text
	replaceValue := func(item hclsyntax.ObjectConsItem, newText string) {
		edits = append(edits, textEdit{
			start: item.ValueExpr.Range().Start.Byte,
			end:   item.ValueExpr.Range().End.Byte,
			text:  newText,
		})
	}

	// accumulate new field lines to insert at the repo's closing brace
	var insertLines []string
	indent := "      " // 6 spaces default

	// always-present string fields
	for _, sf := range []struct {
		key string
		val string
	}{
		{"description", cfg.Description},
		{"visibility", cfg.Visibility},
		{"default_branch", cfg.DefaultBranch},
	} {
		if item, ok := fieldMap[sf.key]; ok {
			old, _ := exprToString(item.ValueExpr)
			if old != sf.val {
				replaceValue(item, fmt.Sprintf("%q", sf.val))
			}
		}
	}

	// always-present bool: has_issues
	if item, ok := fieldMap["has_issues"]; ok {
		old, _ := exprToBool(item.ValueExpr)
		if old != cfg.HasIssues {
			replaceValue(item, boolStr(cfg.HasIssues))
		}
	}

	// optional bools
	for _, ob := range []struct {
		key string
		val bool
	}{
		{"has_wiki", cfg.HasWiki},
		{"allow_auto_merge", cfg.AllowAutoMerge},
		{"allow_update_branch", cfg.AllowUpdateBranch},
		{"delete_branch_on_merge", cfg.DeleteBranchOnMerge},
		{"has_discussions", cfg.HasDiscussions},
		{"has_projects", cfg.HasProjects},
	} {
		item, exists := fieldMap[ob.key]
		if exists && !ob.val {
			old, _ := exprToBool(item.ValueExpr)
			if old {
				edits = append(edits, removeFieldEdit(src, item))
			}
		} else if exists && ob.val {
			old, _ := exprToBool(item.ValueExpr)
			if !old {
				replaceValue(item, boolStr(ob.val))
			}
		} else if !exists && ob.val {
			insertLines = append(insertLines, fmt.Sprintf("%s%-22s = %s\n", indent, ob.key, boolStr(ob.val)))
		}
	}

	// optional string: homepage_url
	if item, exists := fieldMap["homepage_url"]; exists {
		if cfg.HomepageURL == "" {
			edits = append(edits, removeFieldEdit(src, item))
		} else {
			old, _ := exprToString(item.ValueExpr)
			if old != cfg.HomepageURL {
				replaceValue(item, fmt.Sprintf("%q", cfg.HomepageURL))
			}
		}
	} else if cfg.HomepageURL != "" {
		insertLines = append(insertLines, fmt.Sprintf("%s%-22s = %q\n", indent, "homepage_url", cfg.HomepageURL))
	}

	// topics
	if item, exists := fieldMap["topics"]; exists {
		if len(cfg.Topics) > 0 {
			replaceValue(item, formatTopics(cfg.Topics))
		} else {
			edits = append(edits, removeFieldEdit(src, item))
		}
	} else if len(cfg.Topics) > 0 {
		insertLines = append(insertLines, fmt.Sprintf("%s%-22s = %s\n", indent, "topics", formatTopics(cfg.Topics)))
	}

	// team_access — only replace when value actually changed
	if item, exists := fieldMap["team_access"]; exists {
		oldTeams := extractTeamAccessFromItem(item)
		if !teamMapsEqual(oldTeams, cfg.TeamAccess) {
			replaceValue(item, formatTeamAccess(cfg.TeamAccess))
		}
	}

	// branch_protection
	if item, exists := fieldMap["branch_protection"]; exists {
		if cfg.EnableBranchProtection {
			replaceValue(item, formatBranchProtection(cfg))
		} else {
			replaceValue(item, "null")
		}
	}

	// combine all insertions into a single edit at the insert point
	if len(insertLines) > 0 {
		insertAt := addFieldInsertPoint(src, repoObj)
		edits = append(edits, textEdit{start: insertAt, end: insertAt, text: strings.Join(insertLines, "")})
	}

	// sort edits by start descending to apply in reverse order
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

// findReposClosingBrace scans the source lines to find the byte offset of the
// closing brace line for the repos map. It tracks brace depth starting from
// the "repos = {" line.
func findReposClosingBrace(src []byte) (int, error) {
	lines := bytes.Split(src, []byte("\n"))
	inRepos := false
	depth := 0
	offset := 0

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		if !inRepos {
			// look for the repos = { opening
			if bytes.Contains(trimmed, []byte("repos")) && bytes.Contains(trimmed, []byte("= {")) {
				inRepos = true
				depth = 1
				offset += len(line) + 1 // +1 for newline
				continue
			}
		} else {
			// count braces on this line
			for _, b := range line {
				if b == '{' {
					depth++
				} else if b == '}' {
					depth--
					if depth == 0 {
						// this line contains the closing brace of repos
						// return offset pointing to start of this line
						return offset, nil
					}
				}
			}
		}

		offset += len(line) + 1 // +1 for newline
	}

	return 0, fmt.Errorf("repos closing brace not found")
}

var repoTmpl = template.Must(template.New("repo").Parse(`    "{{ .Name }}" = {
      description{{ .DescPad }}= "{{ .Description }}"
      visibility{{ .VisPad }}= "{{ .Visibility }}"
      has_issues{{ .HasIssuesPad }}= {{ .HasIssues }}
{{- if .HasWiki }}
      has_wiki{{ .HasWikiPad }}= {{ .HasWiki }}
{{- end }}
{{- if .HasDiscussions }}
      has_discussions{{ .HasDiscussionsPad }}= {{ .HasDiscussions }}
{{- end }}
{{- if .HasProjects }}
      has_projects{{ .HasProjectsPad }}= {{ .HasProjects }}
{{- end }}
{{- if .HomepageURL }}
      homepage_url{{ .HomepageURLPad }}= "{{ .HomepageURL }}"
{{- end }}
{{- if .AllowAutoMerge }}
      allow_auto_merge{{ .AllowAutoMergePad }}= {{ .AllowAutoMerge }}
{{- end }}
{{- if .AllowUpdateBranch }}
      allow_update_branch{{ .AllowUpdateBranchPad }}= {{ .AllowUpdateBranch }}
{{- end }}
{{- if .DeleteBranchOnMerge }}
      delete_branch_on_merge{{ .DeleteBranchOnMergePad }}= {{ .DeleteBranchOnMerge }}
{{- end }}
      default_branch{{ .DefaultBranchPad }}= "{{ .DefaultBranch }}"
{{- if .Topics }}
      topics{{ .TopicsPad }}= [{{ .TopicsList }}]
{{- end }}
      team_access{{ .TeamAccessPad }}= { {{ .TeamAccessMap }} }
{{- if .BranchProtection }}
      branch_protection = {
        required_reviews                = {{ .RequiredReviews }}
        dismiss_stale_reviews           = {{ .DismissStaleReviews }}
        require_linear_history          = {{ .RequireLinearHistory }}
        require_conversation_resolution = {{ .RequireConversationResolution }}
      }
{{- else }}
      branch_protection{{ .BranchProtPad }}= null
{{- end }}
    }
`))

// repoTmplData holds computed values for the repo template.
type repoTmplData struct {
	Name                          string
	Description                   string
	Visibility                    string
	HasIssues                     string
	HasWiki                       string
	HasDiscussions                string
	HasProjects                   string
	HomepageURL                   string
	AllowAutoMerge                string
	AllowUpdateBranch             string
	DeleteBranchOnMerge           string
	DefaultBranch                 string
	TopicsList                    string
	TeamAccessMap                 string
	Topics                        bool
	BranchProtection              bool
	RequiredReviews               int
	DismissStaleReviews           string
	RequireLinearHistory          string
	RequireConversationResolution string

	// padding for alignment
	DescPad                string
	VisPad                 string
	HasIssuesPad           string
	HasWikiPad             string
	HasDiscussionsPad      string
	HasProjectsPad         string
	HomepageURLPad         string
	AllowAutoMergePad      string
	AllowUpdateBranchPad   string
	DeleteBranchOnMergePad string
	DefaultBranchPad       string
	TopicsPad              string
	TeamAccessPad          string
	BranchProtPad          string
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// pad returns spaces to align "=" at a consistent column.
// targetLen is the longest attribute name length in the block (for alignment).
func pad(attrLen, targetLen int) string {
	n := targetLen - attrLen
	if n < 1 {
		n = 1
	}
	return strings.Repeat(" ", n)
}

func renderRepoEntry(repo conversation.RepoConfig) (string, error) {
	// determine longest attribute name for padding
	attrs := []string{"description", "visibility", "has_issues", "default_branch", "team_access", "branch_protection"}
	if repo.HasWiki {
		attrs = append(attrs, "has_wiki")
	}
	if repo.AllowAutoMerge {
		attrs = append(attrs, "allow_auto_merge")
	}
	if repo.AllowUpdateBranch {
		attrs = append(attrs, "allow_update_branch")
	}
	if repo.DeleteBranchOnMerge {
		attrs = append(attrs, "delete_branch_on_merge")
	}
	if repo.HasDiscussions {
		attrs = append(attrs, "has_discussions")
	}
	if repo.HasProjects {
		attrs = append(attrs, "has_projects")
	}
	if repo.HomepageURL != "" {
		attrs = append(attrs, "homepage_url")
	}
	if len(repo.Topics) > 0 {
		attrs = append(attrs, "topics")
	}

	maxLen := 0
	for _, a := range attrs {
		if len(a) > maxLen {
			maxLen = len(a)
		}
	}

	// build topics list
	var topicParts []string
	for _, t := range repo.Topics {
		topicParts = append(topicParts, fmt.Sprintf("%q", t))
	}
	topicsList := strings.Join(topicParts, ", ")

	// build team_access map
	var teamParts []string
	// sort keys for deterministic output
	var teamKeys []string
	for k := range repo.TeamAccess {
		teamKeys = append(teamKeys, k)
	}
	sort.Strings(teamKeys)
	for _, k := range teamKeys {
		teamParts = append(teamParts, fmt.Sprintf("%q = %q", k, repo.TeamAccess[k]))
	}
	teamAccessMap := strings.Join(teamParts, ", ")

	data := repoTmplData{
		Name:               repo.Name,
		Description:        repo.Description,
		Visibility:         repo.Visibility,
		HasIssues:          boolStr(repo.HasIssues),
		DefaultBranch:      repo.DefaultBranch,
		TopicsList:         topicsList,
		TeamAccessMap:      teamAccessMap,
		Topics:             len(repo.Topics) > 0,
		BranchProtection:   repo.EnableBranchProtection,
		RequiredReviews:    repo.RequiredReviews,
		DismissStaleReviews: boolStr(repo.DismissStaleReviews),
		RequireLinearHistory: boolStr(repo.RequireLinearHistory),
		RequireConversationResolution: boolStr(repo.RequireConversationResolution),

		DescPad:          pad(len("description"), maxLen),
		VisPad:           pad(len("visibility"), maxLen),
		HasIssuesPad:     pad(len("has_issues"), maxLen),
		DefaultBranchPad: pad(len("default_branch"), maxLen),
		TopicsPad:        pad(len("topics"), maxLen),
		TeamAccessPad:    pad(len("team_access"), maxLen),
		BranchProtPad:    pad(len("branch_protection"), maxLen),
	}

	if repo.HasWiki {
		data.HasWiki = boolStr(repo.HasWiki)
		data.HasWikiPad = pad(len("has_wiki"), maxLen)
	}
	if repo.AllowAutoMerge {
		data.AllowAutoMerge = boolStr(repo.AllowAutoMerge)
		data.AllowAutoMergePad = pad(len("allow_auto_merge"), maxLen)
	}
	if repo.AllowUpdateBranch {
		data.AllowUpdateBranch = boolStr(repo.AllowUpdateBranch)
		data.AllowUpdateBranchPad = pad(len("allow_update_branch"), maxLen)
	}
	if repo.DeleteBranchOnMerge {
		data.DeleteBranchOnMerge = boolStr(repo.DeleteBranchOnMerge)
		data.DeleteBranchOnMergePad = pad(len("delete_branch_on_merge"), maxLen)
	}
	if repo.HasDiscussions {
		data.HasDiscussions = boolStr(repo.HasDiscussions)
		data.HasDiscussionsPad = pad(len("has_discussions"), maxLen)
	}
	if repo.HasProjects {
		data.HasProjects = boolStr(repo.HasProjects)
		data.HasProjectsPad = pad(len("has_projects"), maxLen)
	}
	if repo.HomepageURL != "" {
		data.HomepageURL = repo.HomepageURL
		data.HomepageURLPad = pad(len("homepage_url"), maxLen)
	}

	var buf bytes.Buffer
	if err := repoTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
