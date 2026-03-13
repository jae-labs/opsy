package slack

import (
	"strings"
	"testing"

	"github.com/jae-labs/opsy/internal/conversation"
	goslack "github.com/slack-go/slack"
)

// helper to build a values map from field specs
func buildValues(typ string, fields map[string]string) map[string]map[string]goslack.BlockAction {
	values := make(map[string]map[string]goslack.BlockAction)

	// type is always present as a selected option
	values[BlockDnsType] = map[string]goslack.BlockAction{
		ElemDnsType: {SelectedOption: goslack.OptionBlockObject{Value: typ}},
	}

	for blockID, val := range fields {
		switch blockID {
		case BlockDnsName:
			values[BlockDnsName] = map[string]goslack.BlockAction{
				ElemDnsName: {Value: val},
			}
		case BlockDnsContent:
			values[BlockDnsContent] = map[string]goslack.BlockAction{
				ElemDnsContent: {Value: val},
			}
		case BlockDnsPriority:
			values[BlockDnsPriority] = map[string]goslack.BlockAction{
				ElemDnsPriority: {Value: val},
			}
		case BlockDnsProxied:
			if val == "true" {
				values[BlockDnsProxied] = map[string]goslack.BlockAction{
					ElemDnsProxied: {SelectedOptions: []goslack.OptionBlockObject{{Value: "proxied"}}},
				}
			} else {
				values[BlockDnsProxied] = map[string]goslack.BlockAction{
					ElemDnsProxied: {SelectedOptions: nil},
				}
			}
		}
	}

	// ensure name block exists (most tests need it)
	if _, ok := values[BlockDnsName]; !ok {
		values[BlockDnsName] = map[string]goslack.BlockAction{
			ElemDnsName: {Value: "example.com"},
		}
	}
	// ensure content block exists
	if _, ok := values[BlockDnsContent]; !ok {
		values[BlockDnsContent] = map[string]goslack.BlockAction{
			ElemDnsContent: {Value: ""},
		}
	}

	return values
}

func TestValidateDnsFields(t *testing.T) {
	tests := []struct {
		name      string
		typ       string
		fields    map[string]string
		wantErr   map[string]bool // block IDs expected to have errors
		wantClean bool            // true = expect zero errors
	}{
		{
			name: "valid A record",
			typ:  "A",
			fields: map[string]string{
				BlockDnsName:    "app.example.com",
				BlockDnsContent: "192.168.1.1",
			},
			wantClean: true,
		},
		{
			name: "A record with hostname content",
			typ:  "A",
			fields: map[string]string{
				BlockDnsName:    "app.example.com",
				BlockDnsContent: "example.com",
			},
			wantErr: map[string]bool{BlockDnsContent: true},
		},
		{
			name: "valid AAAA record",
			typ:  "AAAA",
			fields: map[string]string{
				BlockDnsName:    "v6.example.com",
				BlockDnsContent: "2001:db8::1",
			},
			wantClean: true,
		},
		{
			name: "AAAA record with IPv4",
			typ:  "AAAA",
			fields: map[string]string{
				BlockDnsName:    "v6.example.com",
				BlockDnsContent: "192.168.1.1",
			},
			wantErr: map[string]bool{BlockDnsContent: true},
		},
		{
			name: "CNAME with IP address",
			typ:  "CNAME",
			fields: map[string]string{
				BlockDnsName:    "www.example.com",
				BlockDnsContent: "10.0.0.1",
			},
			wantErr: map[string]bool{BlockDnsContent: true},
		},
		{
			name: "valid CNAME record",
			typ:  "CNAME",
			fields: map[string]string{
				BlockDnsName:    "www.example.com",
				BlockDnsContent: "origin.example.com",
			},
			wantClean: true,
		},
		{
			name: "MX missing priority",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:    "example.com",
				BlockDnsContent: "mail.example.com",
			},
			wantErr: map[string]bool{BlockDnsPriority: true},
		},
		{
			name: "MX with IP content",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:     "example.com",
				BlockDnsContent:  "10.0.0.1",
				BlockDnsPriority: "10",
			},
			wantErr: map[string]bool{BlockDnsContent: true},
		},
		{
			name: "valid MX record",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:     "example.com",
				BlockDnsContent:  "mail.example.com",
				BlockDnsPriority: "10",
			},
			wantClean: true,
		},
		{
			name: "TXT freeform",
			typ:  "TXT",
			fields: map[string]string{
				BlockDnsName:    "example.com",
				BlockDnsContent: "v=spf1 include:_spf.google.com ~all",
			},
			wantClean: true,
		},
		{
			name: "empty name",
			typ:  "A",
			fields: map[string]string{
				BlockDnsName:    "",
				BlockDnsContent: "192.168.1.1",
			},
			wantErr: map[string]bool{BlockDnsName: true},
		},
		{
			name: "MX priority zero",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:     "example.com",
				BlockDnsContent:  "mail.example.com",
				BlockDnsPriority: "0",
			},
			wantErr: map[string]bool{BlockDnsPriority: true},
		},
		{
			name: "MX priority non-integer",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:     "example.com",
				BlockDnsContent:  "mail.example.com",
				BlockDnsPriority: "abc",
			},
			wantErr: map[string]bool{BlockDnsPriority: true},
		},
		{
			name: "MX proxied warning",
			typ:  "MX",
			fields: map[string]string{
				BlockDnsName:     "example.com",
				BlockDnsContent:  "mail.example.com",
				BlockDnsPriority: "10",
				BlockDnsProxied:  "true",
			},
			wantErr: map[string]bool{BlockDnsProxied: true},
		},
		{
			name: "TXT proxied warning",
			typ:  "TXT",
			fields: map[string]string{
				BlockDnsName:    "example.com",
				BlockDnsContent: "v=spf1 ~all",
				BlockDnsProxied: "true",
			},
			wantErr: map[string]bool{BlockDnsProxied: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := buildValues(tt.typ, tt.fields)
			errs := validateDnsFields(vals)

			if tt.wantClean {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %v", errs)
				}
				return
			}

			for blockID := range tt.wantErr {
				if _, ok := errs[blockID]; !ok {
					t.Errorf("expected error on block %q, got none (errors: %v)", blockID, errs)
				}
			}

			// no unexpected errors
			for blockID := range errs {
				if !tt.wantErr[blockID] {
					t.Errorf("unexpected error on block %q: %s", blockID, errs[blockID])
				}
			}
		})
	}
}

func TestGenerateDnsRecordKey(t *testing.T) {
	tests := []struct {
		name       string
		dnsName    string
		typ        string
		existing   []string
		wantPrefix string
	}{
		{
			name:       "simple subdomain",
			dnsName:    "app.example.com",
			typ:        "A",
			wantPrefix: "app-example-com-a-",
		},
		{
			name:       "at sign falls back to record",
			dnsName:    "@",
			typ:        "A",
			wantPrefix: "record-a-",
		},
		{
			name:       "MX record",
			dnsName:    "example.com",
			typ:        "MX",
			wantPrefix: "example-com-mx-",
		},
		{
			name:       "uppercase in name gets lowered",
			dnsName:    "WWW.Example.Com",
			typ:        "CNAME",
			wantPrefix: "www-example-com-cname-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDnsRecordKey(tt.dnsName, tt.typ, tt.existing)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("generateDnsRecordKey(%q, %q, %v) = %q, want prefix %q",
					tt.dnsName, tt.typ, tt.existing, got, tt.wantPrefix)
			}
			// suffix should be 6 hex chars
			suffix := strings.TrimPrefix(got, tt.wantPrefix)
			if len(suffix) != 6 {
				t.Errorf("expected 6-char hex suffix, got %q", suffix)
			}
		})
	}
}

func TestGenerateDnsRecordKey_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		key := generateDnsRecordKey("test", "A", nil)
		if _, dup := seen[key]; dup {
			t.Fatalf("duplicate key on iteration %d: %s", i, key)
		}
		seen[key] = struct{}{}
	}
}

func TestGenerateDnsRecordKey_AvoidsExisting(t *testing.T) {
	// generate a key, then pass it as existing — next key must differ
	first := generateDnsRecordKey("app", "A", nil)
	second := generateDnsRecordKey("app", "A", []string{first})
	if first == second {
		t.Errorf("second key should differ from first: both %q", first)
	}
}

func TestCheckDnsConflict(t *testing.T) {
	existing := []conversation.DnsConfig{
		{RecordKey: "web-a", Type: "A", Name: "app.example.com"},
		{RecordKey: "mail-mx", Type: "MX", Name: "example.com"},
		{RecordKey: "www-cname", Type: "CNAME", Name: "www.example.com"},
	}

	tests := []struct {
		name    string
		dnsName string
		typ     string
		wantErr bool
	}{
		{
			name:    "A on new name, no conflict",
			dnsName: "api.example.com",
			typ:     "A",
			wantErr: false,
		},
		{
			name:    "second A on same name, no conflict",
			dnsName: "app.example.com",
			typ:     "A",
			wantErr: false,
		},
		{
			name:    "CNAME on name with existing A",
			dnsName: "app.example.com",
			typ:     "CNAME",
			wantErr: true,
		},
		{
			name:    "A on name with existing CNAME",
			dnsName: "www.example.com",
			typ:     "A",
			wantErr: true,
		},
		{
			name:    "AAAA on name with existing CNAME",
			dnsName: "www.example.com",
			typ:     "AAAA",
			wantErr: true,
		},
		{
			name:    "CNAME on name with existing CNAME",
			dnsName: "www.example.com",
			typ:     "CNAME",
			wantErr: true,
		},
		{
			name:    "MX on name with existing MX, no conflict",
			dnsName: "example.com",
			typ:     "MX",
			wantErr: false,
		},
		{
			name:    "TXT on name with existing A, no conflict",
			dnsName: "app.example.com",
			typ:     "TXT",
			wantErr: false,
		},
		{
			name:    "case insensitive match",
			dnsName: "APP.EXAMPLE.COM",
			typ:     "CNAME",
			wantErr: true,
		},
		{
			name:    "no existing records",
			dnsName: "new.example.com",
			typ:     "CNAME",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkDnsConflict(tt.dnsName, tt.typ, existing)
			if tt.wantErr && msg == "" {
				t.Error("expected conflict error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("expected no conflict, got: %s", msg)
			}
		})
	}
}

func TestCheckDnsRecordExists(t *testing.T) {
	keys := []string{"web-a", "mail-mx", "www-cname"}

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "exists", key: "web-a", wantErr: false},
		{name: "exists last", key: "www-cname", wantErr: false},
		{name: "missing", key: "gone-record", wantErr: true},
		{name: "empty key", key: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := checkDnsRecordExists(tt.key, keys)
			if tt.wantErr && msg == "" {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("expected no error, got: %s", msg)
			}
		})
	}
}
