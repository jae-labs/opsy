package conversation

// OrgConfig holds all parameters for the GitHub organization settings block.
type OrgConfig struct {
	Name                     string
	BillingEmail             string
	Blog                     string
	Description              string
	Location                 string
	MembersCanCreateRepos    bool
	DefaultRepoPermission    string // "read", "write", "admin", "none"
	WebCommitSignoffRequired bool
	DependabotAlerts         bool
	DependabotSecurityUpdates bool
	DependencyGraph          bool
}
