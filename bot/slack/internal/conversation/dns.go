package conversation

// DnsConfig holds all parameters collected during the DNS record wizard.
type DnsConfig struct {
	RecordKey string
	Type      string // A, AAAA, CNAME, MX, TXT
	Name      string
	Content   string
	Proxied   bool // only rendered for A/AAAA/CNAME
	Priority  int  // only rendered for MX
	Comment   string
}
