package github

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

func BranchName(repoName string) string {
	sanitized := strings.ToLower(repoName)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = nonAlphaNum.ReplaceAllString(sanitized, "")
	suffix := time.Now().Format("20060102-150405")
	return fmt.Sprintf("opsy/add-repo-%s-%s", sanitized, suffix)
}

func DeleteBranchName(repoName string) string {
	sanitized := strings.ToLower(repoName)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = nonAlphaNum.ReplaceAllString(sanitized, "")
	suffix := time.Now().Format("20060102-150405")
	return fmt.Sprintf("opsy/delete-repo-%s-%s", sanitized, suffix)
}

func BuildDeletePRDescription(repoName, requester, justification string) string {
	return fmt.Sprintf(`## Remove repository: %s

### Justification
%s

### Requested by
Slack user: %s

### Changes
- Removes repository entry from `+"`iac/terraform/github/locals_repos.tf`"+`

---
*Created by Opsy*`, repoName, justification, requester)
}

func SettingsBranchName(repoName string) string {
	sanitized := strings.ToLower(repoName)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = nonAlphaNum.ReplaceAllString(sanitized, "")
	suffix := time.Now().Format("20060102-150405")
	return fmt.Sprintf("opsy/update-repo-%s-%s", sanitized, suffix)
}

func BuildSettingsPRDescription(repoName, requester, justification string) string {
	return fmt.Sprintf(`## Update repository settings: %s

### Justification
%s

### Requested by
Slack user: %s

### Changes
- Updates repository settings in `+"`iac/terraform/github/locals_repos.tf`"+`

---
*Created by Opsy*`, repoName, justification, requester)
}

func DnsBranchName(action, recordKey string) string {
	sanitized := strings.ToLower(recordKey)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = nonAlphaNum.ReplaceAllString(sanitized, "")
	suffix := time.Now().Format("20060102-150405")
	return fmt.Sprintf("opsy/%s-dns-%s-%s", action, sanitized, suffix)
}

func BuildDnsPRDescription(action, zone, recordKey, requester, justification string) string {
	var verb string
	switch action {
	case "add":
		verb = "Add"
	case "delete":
		verb = "Remove"
	case "settings":
		verb = "Update"
	default:
		verb = strings.ToUpper(action[:1]) + action[1:]
	}
	return fmt.Sprintf(`## %s DNS record: %s (zone: %s)

### Justification
%s

### Requested by
Slack user: %s

### Changes
- %ss DNS record entry in `+"`iac/terraform/cloudflare/locals_dns.tf`"+`

---
*Created by Opsy*`, verb, recordKey, zone, justification, requester, verb)
}

func OrgSettingsBranchName() string {
	suffix := time.Now().Format("20060102-150405")
	return fmt.Sprintf("opsy/update-org-settings-%s", suffix)
}

func BuildOrgSettingsPRDescription(requester, justification string) string {
	return fmt.Sprintf(`## Update organization settings

### Justification
%s

### Requested by
Slack user: %s

### Changes
- Updates organization settings in `+"`iac/terraform/github/locals_org.tf`"+`

---
*Created by Opsy*`, justification, requester)
}

func BuildPRDescription(repoName, description, slackUserID, justification string) string {
	return fmt.Sprintf(`## Add repository: %s

%s

### Justification
%s

### Requested by
Slack user: %s

### Changes
- Adds new repository entry to `+"`iac/terraform/github/locals_repos.tf`"+`

---
*Created by Opsy*`, repoName, description, justification, slackUserID)
}
