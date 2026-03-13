package hcl

import (
	"os"
	"strings"
	"testing"

	"github.com/jae-labs/opsy/internal/conversation"
)

func readCloudflareTestdata(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("testdata/locals_dns.tf")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	return src
}

func TestExistingZones(t *testing.T) {
	src := readCloudflareTestdata(t)

	zones, err := ExistingZones(src)
	if err != nil {
		t.Fatalf("ExistingZones: %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if zones[0] != "justanother.engineer" {
		t.Errorf("zone = %q, want justanother.engineer", zones[0])
	}
}

func TestExistingDnsRecordKeys(t *testing.T) {
	src := readCloudflareTestdata(t)

	keys, err := ExistingDnsRecordKeys(src, "justanother.engineer")
	if err != nil {
		t.Fatalf("ExistingDnsRecordKeys: %v", err)
	}
	if len(keys) != 16 {
		t.Fatalf("expected 16 record keys, got %d: %v", len(keys), keys)
	}

	// spot check a few (sorted)
	expected := []string{"aaaa-test", "api-cname", "blog-cname", "grafana-a", "ha-a", "ha-cname",
		"mail-cname", "mx-zoho-1", "mx-zoho-2", "mx-zoho-3", "root-a",
		"txt-dkim", "txt-spf", "txt-verify", "vpn-a", "www-cname"}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], k)
		}
	}
}

func TestExtractDnsConfig_A(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, err := ExtractDnsConfig(src, "justanother.engineer", "ha-a")
	if err != nil {
		t.Fatalf("ExtractDnsConfig: %v", err)
	}

	if cfg.RecordKey != "ha-a" {
		t.Errorf("RecordKey = %q", cfg.RecordKey)
	}
	if cfg.Type != "A" {
		t.Errorf("Type = %q, want A", cfg.Type)
	}
	if cfg.Name != "ha" {
		t.Errorf("Name = %q, want ha", cfg.Name)
	}
	if cfg.Content != "46.7.7.84" {
		t.Errorf("Content = %q", cfg.Content)
	}
	if !cfg.Proxied {
		t.Error("expected Proxied = true")
	}
	if cfg.Comment != "Home Assistant" {
		t.Errorf("Comment = %q", cfg.Comment)
	}
}

func TestExtractDnsConfig_MX(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, err := ExtractDnsConfig(src, "justanother.engineer", "mx-zoho-1")
	if err != nil {
		t.Fatalf("ExtractDnsConfig: %v", err)
	}

	if cfg.Type != "MX" {
		t.Errorf("Type = %q, want MX", cfg.Type)
	}
	if cfg.Priority != 10 {
		t.Errorf("Priority = %d, want 10", cfg.Priority)
	}
	if cfg.Content != "mx.zoho.eu" {
		t.Errorf("Content = %q", cfg.Content)
	}
	if cfg.Comment != "Zoho Mail primary" {
		t.Errorf("Comment = %q", cfg.Comment)
	}
}

func TestExtractDnsConfig_TXT(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, err := ExtractDnsConfig(src, "justanother.engineer", "txt-spf")
	if err != nil {
		t.Fatalf("ExtractDnsConfig: %v", err)
	}

	if cfg.Type != "TXT" {
		t.Errorf("Type = %q, want TXT", cfg.Type)
	}
	if cfg.Content != "v=spf1 include:zoho.eu ~all" {
		t.Errorf("Content = %q", cfg.Content)
	}
	if cfg.Comment != "SPF record" {
		t.Errorf("Comment = %q", cfg.Comment)
	}
}

func TestExtractDnsConfig_AAAA(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, err := ExtractDnsConfig(src, "justanother.engineer", "aaaa-test")
	if err != nil {
		t.Fatalf("ExtractDnsConfig: %v", err)
	}

	if cfg.Type != "AAAA" {
		t.Errorf("Type = %q, want AAAA", cfg.Type)
	}
	if !cfg.Proxied {
		t.Error("expected Proxied = true")
	}
	if cfg.Comment != "IPv6 test record" {
		t.Errorf("Comment = %q", cfg.Comment)
	}
}

func TestAddDnsRecord(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg := conversation.DnsConfig{
		RecordKey: "staging-a",
		Type:      "A",
		Name:      "staging",
		Content:   "192.168.1.100",
		Proxied:   true,
		Comment:   "Staging server",
	}

	result, err := AddDnsRecord(src, "justanother.engineer", cfg)
	if err != nil {
		t.Fatalf("AddDnsRecord: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, `"staging-a"`) {
		t.Error("output missing new record key")
	}
	if !strings.Contains(output, `"192.168.1.100"`) {
		t.Error("output missing new content")
	}
	if !strings.Contains(output, `"Staging server"`) {
		t.Error("output missing new comment")
	}
	// existing records preserved
	if !strings.Contains(output, `"ha-a"`) {
		t.Error("output missing existing ha-a record")
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestAddDnsRecord_MX(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg := conversation.DnsConfig{
		RecordKey: "mx-backup",
		Type:      "MX",
		Name:      "@",
		Content:   "mx-backup.example.com",
		Priority:  99,
	}

	result, err := AddDnsRecord(src, "justanother.engineer", cfg)
	if err != nil {
		t.Fatalf("AddDnsRecord: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "priority = 99") {
		t.Error("output missing priority field")
	}
	if strings.Contains(output, "proxied") && strings.Contains(output, `"mx-backup"`) {
		// proxied should not be rendered for MX records — check the new block only
		// find the mx-backup entry and verify no proxied
		idx := strings.Index(output, `"mx-backup"`)
		endIdx := strings.Index(output[idx:], "}")
		block := output[idx : idx+endIdx]
		if strings.Contains(block, "proxied") {
			t.Error("MX record should not have proxied field")
		}
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestAddDnsRecord_duplicate(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg := conversation.DnsConfig{
		RecordKey: "ha-a",
		Type:      "A",
		Name:      "ha",
		Content:   "1.2.3.4",
	}

	_, err := AddDnsRecord(src, "justanother.engineer", cfg)
	if err == nil {
		t.Fatal("expected error for duplicate record key")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want 'already exists'", err)
	}
}

func TestRemoveDnsRecord(t *testing.T) {
	src := readCloudflareTestdata(t)

	result, err := RemoveDnsRecord(src, "justanother.engineer", "ha-a")
	if err != nil {
		t.Fatalf("RemoveDnsRecord: %v", err)
	}

	output := string(result)
	if strings.Contains(output, `"ha-a"`) {
		t.Error("output still contains ha-a record")
	}
	// other records preserved
	if !strings.Contains(output, `"grafana-a"`) {
		t.Error("output missing grafana-a record")
	}
	if !strings.Contains(output, `"mx-zoho-1"`) {
		t.Error("output missing mx-zoho-1 record")
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}
}

func TestRemoveDnsRecord_notFound(t *testing.T) {
	src := readCloudflareTestdata(t)

	_, err := RemoveDnsRecord(src, "justanother.engineer", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent record")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}

func TestUpdateDnsRecord_changeContent(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, _ := ExtractDnsConfig(src, "justanother.engineer", "ha-a")
	cfg.Content = "10.0.0.1"

	result, err := UpdateDnsRecord(src, "justanother.engineer", "ha-a", cfg)
	if err != nil {
		t.Fatalf("UpdateDnsRecord: %v", err)
	}

	if !strings.Contains(string(result), `"10.0.0.1"`) {
		t.Error("output missing updated content")
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	// round-trip
	extracted, err := ExtractDnsConfig(result, "justanother.engineer", "ha-a")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted.Content != "10.0.0.1" {
		t.Errorf("Content = %q after round-trip", extracted.Content)
	}
	if extracted.Comment != "Home Assistant" {
		t.Error("comment was not preserved")
	}
}

func TestUpdateDnsRecord_noChanges(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, _ := ExtractDnsConfig(src, "justanother.engineer", "ha-a")

	result, err := UpdateDnsRecord(src, "justanother.engineer", "ha-a", cfg)
	if err != nil {
		t.Fatalf("UpdateDnsRecord: %v", err)
	}

	if string(result) != string(src) {
		t.Error("expected no changes, but output differs from input")
	}
}

func TestUpdateDnsRecord_changeType(t *testing.T) {
	src := readCloudflareTestdata(t)

	// change from A (has proxied) to MX (has priority, no proxied)
	cfg := conversation.DnsConfig{
		RecordKey: "ha-a",
		Type:      "MX",
		Name:      "ha",
		Content:   "mx.example.com",
		Priority:  10,
	}

	result, err := UpdateDnsRecord(src, "justanother.engineer", "ha-a", cfg)
	if err != nil {
		t.Fatalf("UpdateDnsRecord: %v", err)
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	extracted, err := ExtractDnsConfig(result, "justanother.engineer", "ha-a")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted.Type != "MX" {
		t.Errorf("Type = %q, want MX", extracted.Type)
	}
	if extracted.Priority != 10 {
		t.Errorf("Priority = %d, want 10", extracted.Priority)
	}
	if extracted.Proxied {
		t.Error("expected Proxied removed for MX type")
	}
}

func TestUpdateDnsRecord_addComment(t *testing.T) {
	src := readCloudflareTestdata(t)

	// vpn-a has no comment
	cfg, _ := ExtractDnsConfig(src, "justanother.engineer", "vpn-a")
	if cfg.Comment != "" {
		t.Fatalf("expected no comment on vpn-a, got %q", cfg.Comment)
	}
	cfg.Comment = "WireGuard VPN"

	result, err := UpdateDnsRecord(src, "justanother.engineer", "vpn-a", cfg)
	if err != nil {
		t.Fatalf("UpdateDnsRecord: %v", err)
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	extracted, err := ExtractDnsConfig(result, "justanother.engineer", "vpn-a")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted.Comment != "WireGuard VPN" {
		t.Errorf("Comment = %q, want WireGuard VPN", extracted.Comment)
	}
}

func TestUpdateDnsRecord_removeComment(t *testing.T) {
	src := readCloudflareTestdata(t)

	cfg, _ := ExtractDnsConfig(src, "justanother.engineer", "ha-a")
	cfg.Comment = ""

	result, err := UpdateDnsRecord(src, "justanother.engineer", "ha-a", cfg)
	if err != nil {
		t.Fatalf("UpdateDnsRecord: %v", err)
	}

	if _, err := Parse(result); err != nil {
		t.Fatalf("result is not valid HCL: %v", err)
	}

	extracted, err := ExtractDnsConfig(result, "justanother.engineer", "ha-a")
	if err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if extracted.Comment != "" {
		t.Errorf("Comment = %q, want empty", extracted.Comment)
	}
}
