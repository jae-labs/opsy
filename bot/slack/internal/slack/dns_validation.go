package slack

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/jae-labs/opsy/internal/conversation"
	goslack "github.com/slack-go/slack"
)

var (
	hostnameRe    = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	nonAlphaNumRe = regexp.MustCompile(`[^a-z0-9]+`)
)

// validateDnsFields checks DNS modal input values and returns a map of block ID
// to error message. An empty map means all fields are valid.
func validateDnsFields(values map[string]map[string]goslack.BlockAction) map[string]string {
	errs := make(map[string]string)

	// name
	name := values[BlockDnsName][ElemDnsName].Value
	if name == "" {
		errs[BlockDnsName] = "Name is required."
	}

	// type + content
	typ := values[BlockDnsType][ElemDnsType].SelectedOption.Value
	content := values[BlockDnsContent][ElemDnsContent].Value
	if content == "" {
		errs[BlockDnsContent] = "Content is required."
	} else {
		validateDnsContent(typ, content, errs)
	}

	// priority (required for MX)
	if typ == "MX" {
		if priBlock, ok := values[BlockDnsPriority]; ok {
			raw := priBlock[ElemDnsPriority].Value
			if raw == "" {
				errs[BlockDnsPriority] = "Priority is required for MX records."
			} else if n, err := strconv.Atoi(raw); err != nil || n <= 0 {
				errs[BlockDnsPriority] = "Priority must be a positive integer."
			}
		} else {
			errs[BlockDnsPriority] = "Priority is required for MX records."
		}
	}

	// proxied warning for MX/TXT
	if typ == "MX" || typ == "TXT" {
		if proxBlock, ok := values[BlockDnsProxied]; ok {
			if len(proxBlock[ElemDnsProxied].SelectedOptions) > 0 {
				errs[BlockDnsProxied] = fmt.Sprintf("Proxied is not supported for %s records.", typ)
			}
		}
	}

	return errs
}

// validateDnsContent adds content-specific errors to errs keyed by BlockDnsContent.
func validateDnsContent(typ, content string, errs map[string]string) {
	ip := net.ParseIP(content)

	switch typ {
	case "A":
		if ip == nil || ip.To4() == nil {
			errs[BlockDnsContent] = "A record content must be a valid IPv4 address."
		}
	case "AAAA":
		if ip == nil || ip.To4() != nil {
			errs[BlockDnsContent] = "AAAA record content must be a valid IPv6 address."
		}
	case "CNAME", "MX":
		if ip != nil {
			errs[BlockDnsContent] = fmt.Sprintf("%s record content must be a hostname, not an IP address.", typ)
		} else if !hostnameRe.MatchString(content) {
			errs[BlockDnsContent] = fmt.Sprintf("%s record content must be a valid hostname.", typ)
		}
	case "TXT":
		// any non-empty string is valid; emptiness checked by caller
	}
}

// checkDnsConflict returns an error string if a new record of the given name and
// type conflicts with existing records. CNAME is exclusive: it cannot coexist
// with any other record on the same name, and no record can be added alongside
// an existing CNAME.
func checkDnsConflict(name, typ string, existing []conversation.DnsConfig) string {
	for _, r := range existing {
		if !strings.EqualFold(r.Name, name) {
			continue
		}
		if typ == "CNAME" {
			return fmt.Sprintf("A %s record already exists for %q. CNAME cannot coexist with other records on the same name.", r.Type, name)
		}
		if r.Type == "CNAME" {
			return fmt.Sprintf("A CNAME record already exists for %q. Cannot add a %s record alongside CNAME.", name, typ)
		}
	}
	return ""
}

// checkDnsRecordExists returns an error string if the given record key is not
// found in the list of existing keys.
func checkDnsRecordExists(key string, existingKeys []string) string {
	for _, k := range existingKeys {
		if k == key {
			return ""
		}
	}
	return fmt.Sprintf("Record %q no longer exists. It may have been removed already.", key)
}

// generateDnsRecordKey derives a unique terraform map key from the record name,
// type, and a random suffix to guarantee uniqueness.
func generateDnsRecordKey(name, typ string, existingKeys []string) string {
	base := strings.ToLower(name)
	base = nonAlphaNumRe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "record"
	}
	base += "-" + strings.ToLower(typ)

	existing := make(map[string]struct{}, len(existingKeys))
	for _, k := range existingKeys {
		existing[k] = struct{}{}
	}

	suffix := randomHex(3)
	candidate := base + "-" + suffix
	for i := 2; ; i++ {
		if _, taken := existing[candidate]; !taken {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%s-%d", base, suffix, i)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
