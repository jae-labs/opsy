package hcl

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jae-labs/opsy/internal/conversation"
)

// ExistingZones returns zone names from the dns_records map.
func ExistingZones(src []byte) ([]string, error) {
	obj, err := findDnsRecordsAttr(src)
	if err != nil {
		return nil, err
	}

	var zones []string
	for _, item := range obj.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			return nil, fmt.Errorf("read zone name: %w", err)
		}
		zones = append(zones, name)
	}
	sort.Strings(zones)
	return zones, nil
}

// ExistingDnsRecordKeys returns record keys within a zone.
func ExistingDnsRecordKeys(src []byte, zone string) ([]string, error) {
	zoneObj, err := findZoneObject(src, zone)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, item := range zoneObj.Items {
		key, err := exprToString(item.KeyExpr)
		if err != nil {
			return nil, fmt.Errorf("read record key: %w", err)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

// ExtractDnsConfig reads a DNS record from HCL source.
func ExtractDnsConfig(src []byte, zone, key string) (conversation.DnsConfig, error) {
	obj, err := findDnsRecordObject(src, zone, key)
	if err != nil {
		return conversation.DnsConfig{}, err
	}

	cfg := conversation.DnsConfig{RecordKey: key}

	for _, item := range obj.Items {
		fieldName, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		switch fieldName {
		case "type":
			cfg.Type, _ = exprToString(item.ValueExpr)
		case "name":
			cfg.Name, _ = exprToString(item.ValueExpr)
		case "content":
			cfg.Content, _ = exprToString(item.ValueExpr)
		case "proxied":
			cfg.Proxied, _ = exprToBool(item.ValueExpr)
		case "priority":
			cfg.Priority, _ = exprToInt(item.ValueExpr)
		case "comment":
			cfg.Comment, _ = exprToString(item.ValueExpr)
		}
	}

	return cfg, nil
}

// AddDnsRecord inserts a new DNS record entry into a zone.
func AddDnsRecord(src []byte, zone string, cfg conversation.DnsConfig) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	existing, err := ExistingDnsRecordKeys(src, zone)
	if err != nil {
		return nil, fmt.Errorf("read existing records: %w", err)
	}
	for _, k := range existing {
		if k == cfg.RecordKey {
			return nil, fmt.Errorf("DNS record %q already exists in zone %q", cfg.RecordKey, zone)
		}
	}

	entry := renderDnsEntry(cfg)

	offset, err := findZoneClosingBrace(src, zone)
	if err != nil {
		return nil, fmt.Errorf("find zone closing brace: %w", err)
	}

	var result bytes.Buffer
	result.Write(src[:offset])
	result.WriteString(entry)
	result.Write(src[offset:])

	out := result.Bytes()
	if _, err := Parse(out); err != nil {
		return nil, fmt.Errorf("modified HCL is invalid: %w", err)
	}
	return out, nil
}

// RemoveDnsRecord removes a DNS record entry from a zone.
func RemoveDnsRecord(src []byte, zone, key string) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	start, end, err := findDnsRecordRange(src, zone, key)
	if err != nil {
		return nil, err
	}

	// strip trailing newlines
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

// UpdateDnsRecord performs in-place field-level edits on a DNS record.
func UpdateDnsRecord(src []byte, zone, key string, cfg conversation.DnsConfig) ([]byte, error) {
	if _, err := Parse(src); err != nil {
		return nil, fmt.Errorf("invalid input HCL: %w", err)
	}

	recObj, err := findDnsRecordObject(src, zone, key)
	if err != nil {
		return nil, err
	}

	fieldMap := make(map[string]hclsyntax.ObjectConsItem)
	for _, item := range recObj.Items {
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

	var insertLines []string
	indent := "        " // 8 spaces for record fields

	wantProxied, wantPriority := dnsFieldsForType(cfg.Type)

	// type — always present
	if item, ok := fieldMap["type"]; ok {
		old, _ := exprToString(item.ValueExpr)
		if old != cfg.Type {
			replaceValue(item, fmt.Sprintf("%q", cfg.Type))
		}
	}

	// name — always present
	if item, ok := fieldMap["name"]; ok {
		old, _ := exprToString(item.ValueExpr)
		if old != cfg.Name {
			replaceValue(item, fmt.Sprintf("%q", cfg.Name))
		}
	}

	// content — always present
	if item, ok := fieldMap["content"]; ok {
		old, _ := exprToString(item.ValueExpr)
		if old != cfg.Content {
			replaceValue(item, fmt.Sprintf("%q", cfg.Content))
		}
	}

	// proxied — conditional on type
	if item, exists := fieldMap["proxied"]; exists {
		if !wantProxied {
			edits = append(edits, removeFieldEdit(src, item))
		} else {
			old, _ := exprToBool(item.ValueExpr)
			if old != cfg.Proxied {
				replaceValue(item, boolStr(cfg.Proxied))
			}
		}
	} else if wantProxied {
		insertLines = append(insertLines, fmt.Sprintf("%sproxied  = %s\n", indent, boolStr(cfg.Proxied)))
	}

	// priority — conditional on type
	if item, exists := fieldMap["priority"]; exists {
		if !wantPriority {
			edits = append(edits, removeFieldEdit(src, item))
		} else {
			old, _ := exprToInt(item.ValueExpr)
			if old != cfg.Priority {
				replaceValue(item, fmt.Sprintf("%d", cfg.Priority))
			}
		}
	} else if wantPriority && cfg.Priority > 0 {
		insertLines = append(insertLines, fmt.Sprintf("%spriority = %d\n", indent, cfg.Priority))
	}

	// comment — optional
	if item, exists := fieldMap["comment"]; exists {
		if cfg.Comment == "" {
			edits = append(edits, removeFieldEdit(src, item))
		} else {
			old, _ := exprToString(item.ValueExpr)
			if old != cfg.Comment {
				replaceValue(item, fmt.Sprintf("%q", cfg.Comment))
			}
		}
	} else if cfg.Comment != "" {
		insertLines = append(insertLines, fmt.Sprintf("%scomment  = %q\n", indent, cfg.Comment))
	}

	// insert new fields before closing brace
	if len(insertLines) > 0 {
		insertAt := addFieldInsertPoint(src, recObj)
		edits = append(edits, textEdit{start: insertAt, end: insertAt, text: strings.Join(insertLines, "")})
	}

	// sort descending to apply in reverse order
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

// findDnsRecordsAttr locates the dns_records attribute in the locals block.
func findDnsRecordsAttr(src []byte) (*hclsyntax.ObjectConsExpr, error) {
	localsBody, err := localsBlockBody(src)
	if err != nil {
		return nil, err
	}

	attr, ok := localsBody.Attributes["dns_records"]
	if !ok {
		return nil, fmt.Errorf("dns_records attribute not found in locals block")
	}

	obj, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("dns_records is not an object expression")
	}
	return obj, nil
}

// findZoneObject locates a zone's inner object within dns_records.
func findZoneObject(src []byte, zone string) (*hclsyntax.ObjectConsExpr, error) {
	dnsObj, err := findDnsRecordsAttr(src)
	if err != nil {
		return nil, err
	}

	for _, item := range dnsObj.Items {
		name, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		if name != zone {
			continue
		}
		inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			return nil, fmt.Errorf("zone %q value is not an object", zone)
		}
		return inner, nil
	}
	return nil, fmt.Errorf("zone %q not found", zone)
}

// findDnsRecordObject locates a record's inner object within a zone.
func findDnsRecordObject(src []byte, zone, key string) (*hclsyntax.ObjectConsExpr, error) {
	zoneObj, err := findZoneObject(src, zone)
	if err != nil {
		return nil, err
	}

	for _, item := range zoneObj.Items {
		k, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		if k != key {
			continue
		}
		inner, ok := item.ValueExpr.(*hclsyntax.ObjectConsExpr)
		if !ok {
			return nil, fmt.Errorf("record %q value is not an object", key)
		}
		return inner, nil
	}
	return nil, fmt.Errorf("DNS record %q not found in zone %q", key, zone)
}

// findDnsRecordRange returns byte range for a record entry (for removal).
func findDnsRecordRange(src []byte, zone, key string) (int, int, error) {
	zoneObj, err := findZoneObject(src, zone)
	if err != nil {
		return 0, 0, err
	}

	for _, item := range zoneObj.Items {
		k, err := exprToString(item.KeyExpr)
		if err != nil {
			continue
		}
		if k != key {
			continue
		}

		keyStart := item.KeyExpr.Range().Start.Byte
		valEnd := item.ValueExpr.Range().End.Byte

		start := keyStart
		for start > 0 && src[start-1] != '\n' {
			start--
		}

		end := valEnd
		for end < len(src) && src[end] != '\n' {
			end++
		}
		if end < len(src) {
			end++
		}

		return start, end, nil
	}
	return 0, 0, fmt.Errorf("DNS record %q not found in zone %q", key, zone)
}

// findZoneClosingBrace finds the insertion point for new records in a zone.
func findZoneClosingBrace(src []byte, zone string) (int, error) {
	zoneObj, err := findZoneObject(src, zone)
	if err != nil {
		return 0, err
	}

	// walk backwards from zone object end to find its closing '}'
	end := zoneObj.SrcRange.End.Byte
	pos := end - 1
	for pos > 0 && src[pos] != '}' {
		pos--
	}
	// find start of that line
	lineStart := pos
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	return lineStart, nil
}

// renderDnsEntry generates HCL text for a new DNS record entry.
func renderDnsEntry(cfg conversation.DnsConfig) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("      %q = {\n", cfg.RecordKey))
	sb.WriteString(fmt.Sprintf("        type    = %q\n", cfg.Type))
	sb.WriteString(fmt.Sprintf("        name    = %q\n", cfg.Name))
	sb.WriteString(fmt.Sprintf("        content = %q\n", cfg.Content))

	wantProxied, wantPriority := dnsFieldsForType(cfg.Type)
	if wantProxied {
		sb.WriteString(fmt.Sprintf("        proxied = %s\n", boolStr(cfg.Proxied)))
	}
	if wantPriority && cfg.Priority > 0 {
		sb.WriteString(fmt.Sprintf("        priority = %d\n", cfg.Priority))
	}
	if cfg.Comment != "" {
		sb.WriteString(fmt.Sprintf("        comment = %q\n", cfg.Comment))
	}

	sb.WriteString("      }\n")
	return sb.String()
}

// dnsFieldsForType returns which optional fields are relevant for a DNS type.
func dnsFieldsForType(typ string) (proxied, priority bool) {
	switch typ {
	case "A", "AAAA", "CNAME":
		return true, false
	case "MX":
		return false, true
	default:
		return false, false
	}
}
