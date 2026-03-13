package slack

import (
	"strings"

	goslack "github.com/slack-go/slack"
)

// validateOrgSettings checks org settings modal input values and returns a map
// of block ID to error message. An empty map means all fields are valid.
func validateOrgSettings(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	name := values[BlockOrgName][ElemOrgName].Value
	if strings.TrimSpace(name) == "" {
		errs[BlockOrgName] = "Name is required."
	}

	email := values[BlockOrgBilling][ElemOrgBilling].Value
	if strings.TrimSpace(email) == "" {
		errs[BlockOrgBilling] = "Billing email is required."
	} else if !strings.Contains(email, "@") {
		errs[BlockOrgBilling] = "Billing email must contain @."
	}

	blog := values[BlockOrgBlog][ElemOrgBlog].Value
	if blog != "" && !strings.HasPrefix(blog, "http://") && !strings.HasPrefix(blog, "https://") {
		errs[BlockOrgBlog] = "Blog must start with http:// or https://."
	}

	desc := values[BlockOrgDesc][ElemOrgDesc].Value
	if strings.TrimSpace(desc) == "" {
		errs[BlockOrgDesc] = "Description is required."
	}

	return errs
}
