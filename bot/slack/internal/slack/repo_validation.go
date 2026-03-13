package slack

import (
	"fmt"
	"regexp"
	"strconv"

	goslack "github.com/slack-go/slack"
)

var repoNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// validateRepoStep1 validates the first step of the repo creation wizard.
func validateRepoStep1(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	name := values[BlockName][ElemName].Value
	if name == "" {
		errs[BlockName] = "Repository name is required."
	} else if len(name) > 100 {
		errs[BlockName] = "Repository name must be 100 characters or fewer."
	} else if !repoNameRe.MatchString(name) {
		errs[BlockName] = "Must start with a letter or digit. Only letters, digits, hyphens, dots, and underscores allowed."
	} else if name[len(name)-1] == '.' {
		errs[BlockName] = "Repository name cannot end with a dot."
	}

	desc := values[BlockDescription][ElemDescription].Value
	if desc == "" {
		errs[BlockDescription] = "Description is required."
	}

	return errs
}

// validateRepoStep2 validates the second step of the repo creation wizard.
func validateRepoStep2(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	branch := values[BlockDefBranch][ElemDefBranch].Value
	if branch == "" {
		errs[BlockDefBranch] = "Default branch is required."
	} else if err := validateBranchName(branch); err != "" {
		errs[BlockDefBranch] = err
	}

	return errs
}

// validateRepoStep3 validates the third step of the repo creation wizard.
func validateRepoStep3(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	if reviews, ok := values[BlockReviews]; ok {
		raw := reviews[ElemReviews].Value
		if raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 5 {
				errs[BlockReviews] = "Required reviews must be a number between 1 and 5."
			}
		}
	}

	return errs
}

// validateSettingsStep1 validates the first step of the settings update wizard.
func validateSettingsStep1(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	desc := values[BlockDescription][ElemDescription].Value
	if desc == "" {
		errs[BlockDescription] = "Description is required."
	}

	return errs
}

// validateSettingsStep2 validates the second step of the settings update wizard.
func validateSettingsStep2(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	branch := values[BlockDefBranch][ElemDefBranch].Value
	if branch == "" {
		errs[BlockDefBranch] = "Default branch is required."
	} else if err := validateBranchName(branch); err != "" {
		errs[BlockDefBranch] = err
	}

	return errs
}

func validateBranchName(name string) string {
	if name == "" {
		return "Branch name is required."
	}
	for _, bad := range []string{"..", " ", "~", "^", ":", "\\", "?", "*", "["} {
		if contains(name, bad) {
			return fmt.Sprintf("Branch name cannot contain %q.", bad)
		}
	}
	if name[0] == '-' || name[0] == '.' {
		return "Branch name cannot start with a hyphen or dot."
	}
	if name[len(name)-1] == '.' || name[len(name)-1] == '/' {
		return "Branch name cannot end with a dot or slash."
	}
	if name[len(name)-1] == '.' || name == "@" {
		return "Invalid branch name."
	}
	return ""
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
