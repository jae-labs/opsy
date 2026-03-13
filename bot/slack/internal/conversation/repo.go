package conversation

// RepoConfig holds all parameters collected during the repo creation wizard.
type RepoConfig struct {
	Name                          string
	Description                   string
	Visibility                    string
	Topics                        []string
	TeamAccess                    map[string]string
	DefaultBranch                 string
	HasIssues                     bool
	HasWiki                       bool
	HasProjects                   bool
	HasDiscussions                bool
	HomepageURL                   string
	AllowAutoMerge                bool
	AllowUpdateBranch             bool
	DeleteBranchOnMerge           bool
	EnableBranchProtection        bool
	RequiredReviews               int
	DismissStaleReviews           bool
	RequireLinearHistory          bool
	RequireConversationResolution bool
}
