package slack

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jae-labs/opsy/internal/conversation"
	"github.com/slack-go/slack"
)

const (
	ActionCategorySelect = "category_select"
	ActionResourceSelect = "resource_select"
	ActionActionSelect   = "action_select"
	ActionConfirm        = "action_confirm"
	ActionCancel         = "action_cancel"

	CallbackRepoStep1  = "repo_step1"
	CallbackRepoStep2  = "repo_step2"
	CallbackRepoStep3  = "repo_step3"
	CallbackDeleteRepo    = "delete_repo"
	CallbackSelectRepo    = "select_repo"
	CallbackSettingsStep1 = "settings_step1"
	CallbackSettingsStep2 = "settings_step2"
	CallbackSettingsStep3 = "settings_step3"

	CallbackDnsAdd          = "dns_add"
	CallbackDnsRemove       = "dns_remove"
	CallbackDnsSelectRecord = "dns_select_record"
	CallbackDnsUpdate       = "dns_update"

	CallbackOrgSettings = "org_settings"

	BlockName         = "block_name"
	BlockDescription  = "block_description"
	BlockVisibility   = "block_visibility"
	BlockTopics       = "block_topics"
	BlockTeamPrefix   = "block_team_"
	BlockDefBranch    = "block_default_branch"
	BlockProtection   = "block_protection"
	BlockReviews      = "block_reviews"
	BlockDismissStale = "block_dismiss_stale"
	BlockLinear       = "block_linear"
	BlockConvRes      = "block_conv_resolution"
	BlockAutoMerge    = "block_auto_merge"
	BlockUpdateBranch = "block_update_branch"
	BlockDeleteBranch   = "block_delete_branch"
	BlockDiscussions   = "block_discussions"
	BlockProjects      = "block_projects"
	BlockHomepage      = "block_homepage"
	BlockJustification = "block_justification"

	ElemName         = "elem_name"
	ElemDescription  = "elem_description"
	ElemVisibility   = "elem_visibility"
	ElemTopics       = "elem_topics"
	ElemTeamPrefix   = "elem_team_"
	ElemDefBranch    = "elem_default_branch"
	ElemProtection   = "elem_protection"
	ElemReviews      = "elem_reviews"
	ElemDismissStale = "elem_dismiss_stale"
	ElemLinear       = "elem_linear"
	ElemConvRes      = "elem_conv_resolution"
	ElemAutoMerge    = "elem_auto_merge"
	ElemUpdateBranch = "elem_update_branch"
	ElemDeleteBranch     = "elem_delete_branch"
	ElemDiscussions      = "elem_discussions"
	ElemProjects         = "elem_projects"
	ElemHomepage         = "elem_homepage"
	ElemJustification    = "elem_justification"

	BlockDeleteTarget = "block_delete_target"
	ElemDeleteTarget  = "elem_delete_target"

	BlockSelectRepo = "block_select_repo"
	ElemSelectRepo  = "elem_select_repo"

	BlockDnsKey      = "block_dns_key"
	BlockDnsType     = "block_dns_type"
	BlockDnsName     = "block_dns_name"
	BlockDnsContent  = "block_dns_content"
	BlockDnsProxied  = "block_dns_proxied"
	BlockDnsPriority = "block_dns_priority"
	BlockDnsComment  = "block_dns_comment"
	BlockDnsRecord   = "block_dns_record"

	ElemDnsKey      = "elem_dns_key"
	ElemDnsType     = "elem_dns_type"
	ElemDnsName     = "elem_dns_name"
	ElemDnsContent  = "elem_dns_content"
	ElemDnsProxied  = "elem_dns_proxied"
	ElemDnsPriority = "elem_dns_priority"
	ElemDnsComment  = "elem_dns_comment"
	ElemDnsRecord   = "elem_dns_record"

	BlockOrgName          = "block_org_name"
	BlockOrgBilling       = "block_org_billing"
	BlockOrgBlog          = "block_org_blog"
	BlockOrgDesc          = "block_org_desc"
	BlockOrgLocation      = "block_org_location"
	BlockOrgPermission    = "block_org_permission"
	BlockOrgMembersCreate = "block_org_members_create"
	BlockOrgSignoff       = "block_org_signoff"
	BlockOrgDepAlerts     = "block_org_dep_alerts"
	BlockOrgDepSec        = "block_org_dep_sec"
	BlockOrgDepGraph      = "block_org_dep_graph"

	ElemOrgName          = "elem_org_name"
	ElemOrgBilling       = "elem_org_billing"
	ElemOrgBlog          = "elem_org_blog"
	ElemOrgDesc          = "elem_org_desc"
	ElemOrgLocation      = "elem_org_location"
	ElemOrgPermission    = "elem_org_permission"
	ElemOrgMembersCreate = "elem_org_members_create"
	ElemOrgSignoff       = "elem_org_signoff"
	ElemOrgDepAlerts     = "elem_org_dep_alerts"
	ElemOrgDepSec        = "elem_org_dep_sec"
	ElemOrgDepGraph      = "elem_org_dep_graph"
)

// teamRoleOptions defines the available GitHub team permission levels.
// "none" means the team is not assigned to the repo.
var teamRoleOptions = []*slack.OptionBlockObject{
	slack.NewOptionBlockObject("none", slack.NewTextBlockObject("plain_text", "None", false, false), nil),
	slack.NewOptionBlockObject("pull", slack.NewTextBlockObject("plain_text", "Read", false, false), nil),
	slack.NewOptionBlockObject("triage", slack.NewTextBlockObject("plain_text", "Triage", false, false), nil),
	slack.NewOptionBlockObject("push", slack.NewTextBlockObject("plain_text", "Write", false, false), nil),
	slack.NewOptionBlockObject("maintain", slack.NewTextBlockObject("plain_text", "Maintain", false, false), nil),
	slack.NewOptionBlockObject("admin", slack.NewTextBlockObject("plain_text", "Admin", false, false), nil),
}

// teamRoleBlocksDefault builds one dropdown per team, defaulting to "pull" (Read).
func teamRoleBlocksDefault(teams []string) []slack.Block {
	return teamRoleBlocks(teams, nil)
}

// teamRoleBlocksWithConfig builds one dropdown per team, pre-selecting the
// existing role from cfg or "none" if the team is not assigned.
func teamRoleBlocks(teams []string, cfg map[string]string) []slack.Block {
	var blocks []slack.Block
	for _, t := range teams {
		elemID := ElemTeamPrefix + t
		elem := slack.NewOptionsSelectBlockElement("static_select",
			slack.NewTextBlockObject("plain_text", "Select role", false, false),
			elemID, teamRoleOptions...)

		initial := "pull"
		if cfg != nil {
			if role, ok := cfg[t]; ok {
				initial = role
			} else {
				initial = "none"
			}
		}
		for _, opt := range teamRoleOptions {
			if opt.Value == initial {
				elem.InitialOption = opt
				break
			}
		}

		blockID := BlockTeamPrefix + t
		block := slack.NewInputBlock(blockID,
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("Team: %s", t), false, false), nil, elem)
		blocks = append(blocks, block)
	}
	return blocks
}

// parseTeamRoleValues extracts team->role mappings from submission values,
// skipping teams set to "none".
func parseTeamRoleValues(values map[string]map[string]slack.BlockAction, teams []string) map[string]string {
	teamAccess := make(map[string]string)
	for _, t := range teams {
		blockID := BlockTeamPrefix + t
		elemID := ElemTeamPrefix + t
		if block, ok := values[blockID]; ok {
			if elem, ok := block[elemID]; ok {
				role := elem.SelectedOption.Value
				if role != "" && role != "none" {
					teamAccess[t] = role
				}
			}
		}
	}
	return teamAccess
}

// CategoryOption defines a selectable platform/service.
type CategoryOption struct {
	Value string
	Label string
}

// DnsRecordOption pairs a terraform key with a human-readable label.
type DnsRecordOption struct {
	Key   string // terraform map key, used as dropdown value
	Label string // e.g. "app.example.com (A)"
}

// categories is the list of available platforms.
// add new entries here as they become available.
var categories = []CategoryOption{
	{Value: "github", Label: "GitHub"},
	{Value: "cloudflare", Label: "Cloudflare"},
	{Value: "doppler", Label: "Doppler"},
}

// actionOptions maps resource -> available actions.
var actionOptions = map[string][]CategoryOption{
	"repo": {
		{Value: "add", Label: "Add"},
		{Value: "delete", Label: "Remove"},
		{Value: "settings", Label: "Update"},
	},
	"dns": {
		{Value: "add", Label: "Add"},
		{Value: "delete", Label: "Remove"},
		{Value: "settings", Label: "Update"},
	},
}

// resourceOptions maps category -> available resource types.
var resourceOptions = map[string][]CategoryOption{
	"github": {
		{Value: "repo", Label: "Repository"},
		{Value: "user_management", Label: "User Management"},
		{Value: "org_settings", Label: "Org Settings"},
	},
	"cloudflare": {
		{Value: "dns", Label: "DNS Records"},
	},
}

func WelcomeBlocks() []slack.Block {
	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Hey there, I'm Opsy. Let's get things done.\nWhat would you like to set up?\n\n_Each selection is final. If something needs to change or the flow is interrupted, start a new thread._", false, false),
		nil, nil,
	)

	opts := make([]*slack.OptionBlockObject, len(categories))
	for i, c := range categories {
		opts[i] = slack.NewOptionBlockObject(c.Value,
			slack.NewTextBlockObject("plain_text", c.Label, false, false), nil)
	}
	sel := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a platform...", false, false),
		ActionCategorySelect, opts...)
	actions := slack.NewActionBlock("welcome_actions", sel)

	return []slack.Block{header, actions}
}

func ResourceBlocks(category string) []slack.Block {
	resources, ok := resourceOptions[category]
	if !ok {
		return ComingSoonBlocks(category)
	}

	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* it is.\nWhat kind of resource are we working with?", category), false, false),
		nil, nil,
	)

	opts := make([]*slack.OptionBlockObject, len(resources))
	for i, r := range resources {
		opts[i] = slack.NewOptionBlockObject(r.Value,
			slack.NewTextBlockObject("plain_text", r.Label, false, false), nil)
	}
	sel := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a resource...", false, false),
		ActionResourceSelect, opts...)
	actions := slack.NewActionBlock("resource_actions", sel)

	return []slack.Block{header, actions}
}

func ComingSoonBlocks(resource string) []slack.Block {
	text := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* is not supported yet. Stay tuned.\nSend a new message when you need me again.", resource), false, false),
		nil, nil,
	)
	return []slack.Block{text}
}

func RepoStep1Modal() slack.ModalViewRequest {
	nameElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "my-new-repo", false, false), ElemName)
	nameBlock := slack.NewInputBlock(BlockName,
		slack.NewTextBlockObject("plain_text", "Repository Name", false, false), nil, nameElem)

	descElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "A brief description...", false, false), ElemDescription)
	descBlock := slack.NewInputBlock(BlockDescription,
		slack.NewTextBlockObject("plain_text", "Description", false, false), nil, descElem)

	visOpts := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject("public", slack.NewTextBlockObject("plain_text", "Public", false, false), nil),
		slack.NewOptionBlockObject("private", slack.NewTextBlockObject("plain_text", "Private", false, false), nil),
	}
	visElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select visibility", false, false),
		ElemVisibility, visOpts...)
	visBlock := slack.NewInputBlock(BlockVisibility,
		slack.NewTextBlockObject("plain_text", "Visibility", false, false), nil, visElem)

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why is this repository needed?", false, false), ElemJustification)
	justElem.WithMultiline(true)
	justElem.WithMinLength(20)
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "New Repo (1/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackRepoStep1,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{nameBlock, descBlock, visBlock, justBlock},
		},
	}
}

func RepoStep2Modal(existingTeams []string) slack.ModalViewRequest {
	topicsElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "golang, api", false, false), ElemTopics)
	topicsBlock := slack.NewInputBlock(BlockTopics,
		slack.NewTextBlockObject("plain_text", "Topics", false, false),
		slack.NewTextBlockObject("plain_text", "Comma-separated, e.g. golang, api, cli", false, false), topicsElem)
	topicsBlock.Optional = true

	blocks := []slack.Block{topicsBlock}

	blocks = append(blocks, teamRoleBlocksDefault(existingTeams)...)

	branchElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "main", false, false), ElemDefBranch)
	branchElem.InitialValue = "main"
	branchBlock := slack.NewInputBlock(BlockDefBranch,
		slack.NewTextBlockObject("plain_text", "Default Branch", false, false), nil, branchElem)
	blocks = append(blocks, branchBlock)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "New Repo (2/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackRepoStep2,
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
	}
}

func RepoStep3Modal() slack.ModalViewRequest {
	protOpt := slack.NewOptionBlockObject("enabled",
		slack.NewTextBlockObject("plain_text", "Enable branch protection", false, false), nil)
	protElem := slack.NewCheckboxGroupsBlockElement(ElemProtection, protOpt)
	protBlock := slack.NewInputBlock(BlockProtection,
		slack.NewTextBlockObject("plain_text", "Branch Protection", false, false), nil, protElem)
	protBlock.Optional = true

	reviewsElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "1", false, false), ElemReviews)
	reviewsElem.InitialValue = "1"
	reviewsBlock := slack.NewInputBlock(BlockReviews,
		slack.NewTextBlockObject("plain_text", "Required Reviews (1-5)", false, false),
		slack.NewTextBlockObject("plain_text", "Only applies if branch protection enabled", false, false), reviewsElem)
	reviewsBlock.Optional = true

	dismissOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Dismiss stale reviews on new push", false, false), nil)
	dismissElem := slack.NewCheckboxGroupsBlockElement(ElemDismissStale, dismissOpt)
	dismissBlock := slack.NewInputBlock(BlockDismissStale,
		slack.NewTextBlockObject("plain_text", "Dismiss Stale Reviews?", false, false), nil, dismissElem)
	dismissBlock.Optional = true

	linearOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Require linear commit history", false, false), nil)
	linearElem := slack.NewCheckboxGroupsBlockElement(ElemLinear, linearOpt)
	linearBlock := slack.NewInputBlock(BlockLinear,
		slack.NewTextBlockObject("plain_text", "Require Linear History?", false, false), nil, linearElem)
	linearBlock.Optional = true

	convOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "All conversations must be resolved", false, false), nil)
	convElem := slack.NewCheckboxGroupsBlockElement(ElemConvRes, convOpt)
	convBlock := slack.NewInputBlock(BlockConvRes,
		slack.NewTextBlockObject("plain_text", "Require Conversation Resolution?", false, false), nil, convElem)
	convBlock.Optional = true

	divider := slack.NewDividerBlock()

	autoMergeOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Allow auto merge", false, false), nil)
	autoMergeElem := slack.NewCheckboxGroupsBlockElement(ElemAutoMerge, autoMergeOpt)
	autoMergeBlock := slack.NewInputBlock(BlockAutoMerge,
		slack.NewTextBlockObject("plain_text", "Allow Auto Merge?", false, false), nil, autoMergeElem)
	autoMergeBlock.Optional = true

	updateOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Allow update branch button", false, false), nil)
	updateElem := slack.NewCheckboxGroupsBlockElement(ElemUpdateBranch, updateOpt)
	updateBlock := slack.NewInputBlock(BlockUpdateBranch,
		slack.NewTextBlockObject("plain_text", "Allow Update Branch?", false, false), nil, updateElem)
	updateBlock.Optional = true

	deleteOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Delete head branch after merge", false, false), nil)
	deleteElem := slack.NewCheckboxGroupsBlockElement(ElemDeleteBranch, deleteOpt)
	deleteBlock := slack.NewInputBlock(BlockDeleteBranch,
		slack.NewTextBlockObject("plain_text", "Delete Branch on Merge?", false, false), nil, deleteElem)
	deleteBlock.Optional = true

	discOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable discussions", false, false), nil)
	discElem := slack.NewCheckboxGroupsBlockElement(ElemDiscussions, discOpt)
	discBlock := slack.NewInputBlock(BlockDiscussions,
		slack.NewTextBlockObject("plain_text", "Enable Discussions?", false, false), nil, discElem)
	discBlock.Optional = true

	projOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable projects", false, false), nil)
	projElem := slack.NewCheckboxGroupsBlockElement(ElemProjects, projOpt)
	projBlock := slack.NewInputBlock(BlockProjects,
		slack.NewTextBlockObject("plain_text", "Enable Projects?", false, false), nil, projElem)
	projBlock.Optional = true

	homepageElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "https://example.com", false, false), ElemHomepage)
	homepageBlock := slack.NewInputBlock(BlockHomepage,
		slack.NewTextBlockObject("plain_text", "Homepage URL", false, false),
		slack.NewTextBlockObject("plain_text", "Optional project homepage", false, false), homepageElem)
	homepageBlock.Optional = true

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "New Repo (3/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackRepoStep3,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				protBlock, reviewsBlock, dismissBlock, linearBlock, convBlock,
				divider,
				autoMergeBlock, updateBlock, deleteBlock,
				discBlock, projBlock, homepageBlock,
			},
		},
	}
}

func ActionBlocks(resource string) []slack.Block {
	actions, ok := actionOptions[resource]
	if !ok {
		return ComingSoonBlocks(resource)
	}

	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Got it. What would you like to do with this resource?", false, false),
		nil, nil,
	)

	opts := make([]*slack.OptionBlockObject, len(actions))
	for i, a := range actions {
		opts[i] = slack.NewOptionBlockObject(a.Value,
			slack.NewTextBlockObject("plain_text", a.Label, false, false), nil)
	}
	sel := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select an action...", false, false),
		ActionActionSelect, opts...)
	actionsBlock := slack.NewActionBlock("action_actions", sel)

	return []slack.Block{header, actionsBlock}
}

// LockedCategoryBlocks replaces the category dropdown with static text after selection.
func LockedCategoryBlocks(categoryLabel string) []slack.Block {
	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Hey there, I'm Opsy. Let's get things done.\nWhat would you like to set up?\n\n_Each selection is final. If something needs to change or the flow is interrupted, start a new thread._", false, false),
		nil, nil,
	)
	selected := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("> Platform: *%s*", categoryLabel), false, false),
		nil, nil,
	)
	return []slack.Block{header, selected}
}

// LockedResourceBlocks replaces the resource dropdown with static text after selection.
func LockedResourceBlocks(category, resourceLabel string) []slack.Block {
	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* it is.\nWhat kind of resource are we working with?", category), false, false),
		nil, nil,
	)
	selected := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("> Resource: *%s*", resourceLabel), false, false),
		nil, nil,
	)
	return []slack.Block{header, selected}
}

// LockedActionBlocks replaces the action dropdown with static text after selection.
func LockedActionBlocks(actionLabel string) []slack.Block {
	header := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Got it. What would you like to do with this resource?", false, false),
		nil, nil,
	)
	selected := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("> Action: *%s*", actionLabel), false, false),
		nil, nil,
	)
	return []slack.Block{header, selected}
}

// labelForValue returns the label for a given value in a CategoryOption slice.
// falls back to value itself if not found.
func labelForValue(options []CategoryOption, value string) string {
	for _, o := range options {
		if o.Value == value {
			return o.Label
		}
	}
	return value
}

func DeleteRepoModal(existingRepos []string) slack.ModalViewRequest {
	repoOpts := make([]*slack.OptionBlockObject, len(existingRepos))
	for i, r := range existingRepos {
		repoOpts[i] = slack.NewOptionBlockObject(r,
			slack.NewTextBlockObject("plain_text", r, false, false), nil)
	}
	repoElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a repository...", false, false),
		ElemDeleteTarget, repoOpts...)
	repoBlock := slack.NewInputBlock(BlockDeleteTarget,
		slack.NewTextBlockObject("plain_text", "Repository to Remove", false, false), nil, repoElem)

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why should this repository be removed?", false, false), ElemJustification)
	justElem.WithMultiline(true)
	justElem.WithMinLength(20)
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Remove Repository", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackDeleteRepo,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{repoBlock, justBlock},
		},
	}
}

func DeleteConfirmationBlocks(repoName, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*You're about to remove a repository. Please confirm.*\n\n")
	sb.WriteString(fmt.Sprintf("*Repository:* `%s`\n", repoName))
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n", justification))

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false),
		nil, nil,
	)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	confirmActions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, confirmActions}
}

func SelectRepoModal(existingRepos []string) slack.ModalViewRequest {
	repoOpts := make([]*slack.OptionBlockObject, len(existingRepos))
	for i, r := range existingRepos {
		repoOpts[i] = slack.NewOptionBlockObject(r,
			slack.NewTextBlockObject("plain_text", r, false, false), nil)
	}
	repoElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a repository...", false, false),
		ElemSelectRepo, repoOpts...)
	repoBlock := slack.NewInputBlock(BlockSelectRepo,
		slack.NewTextBlockObject("plain_text", "Repository", false, false), nil, repoElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Select Repository", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackSelectRepo,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{repoBlock},
		},
	}
}

func SettingsStep1Modal(cfg conversation.RepoConfig) slack.ModalViewRequest {
	nameCtx := slack.NewContextBlock("settings_repo_name",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Repository:* `%s`", cfg.Name), false, false))

	descElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "A brief description...", false, false), ElemDescription)
	descElem.InitialValue = cfg.Description
	descBlock := slack.NewInputBlock(BlockDescription,
		slack.NewTextBlockObject("plain_text", "Description", false, false), nil, descElem)

	visOpts := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject("public", slack.NewTextBlockObject("plain_text", "Public", false, false), nil),
		slack.NewOptionBlockObject("private", slack.NewTextBlockObject("plain_text", "Private", false, false), nil),
	}
	visElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select visibility", false, false),
		ElemVisibility, visOpts...)
	for _, o := range visOpts {
		if o.Value == cfg.Visibility {
			visElem.InitialOption = o
			break
		}
	}
	visBlock := slack.NewInputBlock(BlockVisibility,
		slack.NewTextBlockObject("plain_text", "Visibility", false, false), nil, visElem)

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why are these changes needed?", false, false), ElemJustification)
	justElem.Multiline = true
	justElem.MinLength = 20
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Edit Repo (1/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackSettingsStep1,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{nameCtx, descBlock, visBlock, justBlock},
		},
	}
}

func SettingsStep2Modal(cfg conversation.RepoConfig, teams []string) slack.ModalViewRequest {
	topicsElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "golang, api", false, false), ElemTopics)
	if len(cfg.Topics) > 0 {
		topicsElem.InitialValue = strings.Join(cfg.Topics, ", ")
	}
	topicsBlock := slack.NewInputBlock(BlockTopics,
		slack.NewTextBlockObject("plain_text", "Topics", false, false),
		slack.NewTextBlockObject("plain_text", "Comma-separated, e.g. golang, api, cli", false, false), topicsElem)
	topicsBlock.Optional = true

	blocks := []slack.Block{topicsBlock}

	blocks = append(blocks, teamRoleBlocks(teams, cfg.TeamAccess)...)

	branchElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "main", false, false), ElemDefBranch)
	branchElem.InitialValue = cfg.DefaultBranch
	branchBlock := slack.NewInputBlock(BlockDefBranch,
		slack.NewTextBlockObject("plain_text", "Default Branch", false, false), nil, branchElem)
	blocks = append(blocks, branchBlock)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Edit Repo (2/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackSettingsStep2,
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
	}
}

func SettingsStep3Modal(cfg conversation.RepoConfig) slack.ModalViewRequest {
	protOpt := slack.NewOptionBlockObject("enabled",
		slack.NewTextBlockObject("plain_text", "Enable branch protection", false, false), nil)
	protElem := slack.NewCheckboxGroupsBlockElement(ElemProtection, protOpt)
	if cfg.EnableBranchProtection {
		protElem.InitialOptions = []*slack.OptionBlockObject{protOpt}
	}
	protBlock := slack.NewInputBlock(BlockProtection,
		slack.NewTextBlockObject("plain_text", "Branch Protection", false, false), nil, protElem)
	protBlock.Optional = true

	reviewsElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "1", false, false), ElemReviews)
	if cfg.RequiredReviews > 0 {
		reviewsElem.InitialValue = strconv.Itoa(cfg.RequiredReviews)
	} else {
		reviewsElem.InitialValue = "1"
	}
	reviewsBlock := slack.NewInputBlock(BlockReviews,
		slack.NewTextBlockObject("plain_text", "Required Reviews (1-5)", false, false),
		slack.NewTextBlockObject("plain_text", "Only applies if branch protection enabled", false, false), reviewsElem)
	reviewsBlock.Optional = true

	dismissOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Dismiss stale reviews on new push", false, false), nil)
	dismissElem := slack.NewCheckboxGroupsBlockElement(ElemDismissStale, dismissOpt)
	if cfg.DismissStaleReviews {
		dismissElem.InitialOptions = []*slack.OptionBlockObject{dismissOpt}
	}
	dismissBlock := slack.NewInputBlock(BlockDismissStale,
		slack.NewTextBlockObject("plain_text", "Dismiss Stale Reviews?", false, false), nil, dismissElem)
	dismissBlock.Optional = true

	linearOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Require linear commit history", false, false), nil)
	linearElem := slack.NewCheckboxGroupsBlockElement(ElemLinear, linearOpt)
	if cfg.RequireLinearHistory {
		linearElem.InitialOptions = []*slack.OptionBlockObject{linearOpt}
	}
	linearBlock := slack.NewInputBlock(BlockLinear,
		slack.NewTextBlockObject("plain_text", "Require Linear History?", false, false), nil, linearElem)
	linearBlock.Optional = true

	convOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "All conversations must be resolved", false, false), nil)
	convElem := slack.NewCheckboxGroupsBlockElement(ElemConvRes, convOpt)
	if cfg.RequireConversationResolution {
		convElem.InitialOptions = []*slack.OptionBlockObject{convOpt}
	}
	convBlock := slack.NewInputBlock(BlockConvRes,
		slack.NewTextBlockObject("plain_text", "Require Conversation Resolution?", false, false), nil, convElem)
	convBlock.Optional = true

	divider := slack.NewDividerBlock()

	autoMergeOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Allow auto merge", false, false), nil)
	autoMergeElem := slack.NewCheckboxGroupsBlockElement(ElemAutoMerge, autoMergeOpt)
	if cfg.AllowAutoMerge {
		autoMergeElem.InitialOptions = []*slack.OptionBlockObject{autoMergeOpt}
	}
	autoMergeBlock := slack.NewInputBlock(BlockAutoMerge,
		slack.NewTextBlockObject("plain_text", "Allow Auto Merge?", false, false), nil, autoMergeElem)
	autoMergeBlock.Optional = true

	updateOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Allow update branch button", false, false), nil)
	updateElem := slack.NewCheckboxGroupsBlockElement(ElemUpdateBranch, updateOpt)
	if cfg.AllowUpdateBranch {
		updateElem.InitialOptions = []*slack.OptionBlockObject{updateOpt}
	}
	updateBlock := slack.NewInputBlock(BlockUpdateBranch,
		slack.NewTextBlockObject("plain_text", "Allow Update Branch?", false, false), nil, updateElem)
	updateBlock.Optional = true

	deleteOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Delete head branch after merge", false, false), nil)
	deleteElem := slack.NewCheckboxGroupsBlockElement(ElemDeleteBranch, deleteOpt)
	if cfg.DeleteBranchOnMerge {
		deleteElem.InitialOptions = []*slack.OptionBlockObject{deleteOpt}
	}
	deleteBlock := slack.NewInputBlock(BlockDeleteBranch,
		slack.NewTextBlockObject("plain_text", "Delete Branch on Merge?", false, false), nil, deleteElem)
	deleteBlock.Optional = true

	discOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable discussions", false, false), nil)
	discElem := slack.NewCheckboxGroupsBlockElement(ElemDiscussions, discOpt)
	if cfg.HasDiscussions {
		discElem.InitialOptions = []*slack.OptionBlockObject{discOpt}
	}
	discBlock := slack.NewInputBlock(BlockDiscussions,
		slack.NewTextBlockObject("plain_text", "Enable Discussions?", false, false), nil, discElem)
	discBlock.Optional = true

	projOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable projects", false, false), nil)
	projElem := slack.NewCheckboxGroupsBlockElement(ElemProjects, projOpt)
	if cfg.HasProjects {
		projElem.InitialOptions = []*slack.OptionBlockObject{projOpt}
	}
	projBlock := slack.NewInputBlock(BlockProjects,
		slack.NewTextBlockObject("plain_text", "Enable Projects?", false, false), nil, projElem)
	projBlock.Optional = true

	homepageElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "https://example.com", false, false), ElemHomepage)
	if cfg.HomepageURL != "" {
		homepageElem.InitialValue = cfg.HomepageURL
	}
	homepageBlock := slack.NewInputBlock(BlockHomepage,
		slack.NewTextBlockObject("plain_text", "Homepage URL", false, false),
		slack.NewTextBlockObject("plain_text", "Optional project homepage", false, false), homepageElem)
	homepageBlock.Optional = true

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Edit Repo (3/3)", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackSettingsStep3,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				protBlock, reviewsBlock, dismissBlock, linearBlock, convBlock,
				divider,
				autoMergeBlock, updateBlock, deleteBlock,
				discBlock, projBlock, homepageBlock,
			},
		},
	}
}

func SettingsConfirmationBlocks(repoName string, oldCfg, newCfg conversation.RepoConfig, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*Here are the proposed setting changes. Please review before submitting.*\n\n")
	sb.WriteString(fmt.Sprintf("*Repository:* `%s`\n", repoName))
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n\n", justification))

	changed := false
	if oldCfg.Description != newCfg.Description {
		sb.WriteString(fmt.Sprintf("*Description:* %s -> %s\n", oldCfg.Description, newCfg.Description))
		changed = true
	}
	if oldCfg.Visibility != newCfg.Visibility {
		sb.WriteString(fmt.Sprintf("*Visibility:* %s -> %s\n", oldCfg.Visibility, newCfg.Visibility))
		changed = true
	}
	if oldCfg.DefaultBranch != newCfg.DefaultBranch {
		sb.WriteString(fmt.Sprintf("*Default Branch:* %s -> %s\n", oldCfg.DefaultBranch, newCfg.DefaultBranch))
		changed = true
	}
	if oldCfg.HasIssues != newCfg.HasIssues {
		sb.WriteString(fmt.Sprintf("*Has Issues:* %v -> %v\n", oldCfg.HasIssues, newCfg.HasIssues))
		changed = true
	}
	if oldCfg.HasWiki != newCfg.HasWiki {
		sb.WriteString(fmt.Sprintf("*Has Wiki:* %v -> %v\n", oldCfg.HasWiki, newCfg.HasWiki))
		changed = true
	}
	if oldCfg.AllowAutoMerge != newCfg.AllowAutoMerge {
		sb.WriteString(fmt.Sprintf("*Auto Merge:* %v -> %v\n", oldCfg.AllowAutoMerge, newCfg.AllowAutoMerge))
		changed = true
	}
	if oldCfg.AllowUpdateBranch != newCfg.AllowUpdateBranch {
		sb.WriteString(fmt.Sprintf("*Update Branch:* %v -> %v\n", oldCfg.AllowUpdateBranch, newCfg.AllowUpdateBranch))
		changed = true
	}
	if oldCfg.DeleteBranchOnMerge != newCfg.DeleteBranchOnMerge {
		sb.WriteString(fmt.Sprintf("*Delete Branch on Merge:* %v -> %v\n", oldCfg.DeleteBranchOnMerge, newCfg.DeleteBranchOnMerge))
		changed = true
	}
	if oldCfg.HasDiscussions != newCfg.HasDiscussions {
		sb.WriteString(fmt.Sprintf("*Discussions:* %v -> %v\n", oldCfg.HasDiscussions, newCfg.HasDiscussions))
		changed = true
	}
	if oldCfg.HasProjects != newCfg.HasProjects {
		sb.WriteString(fmt.Sprintf("*Projects:* %v -> %v\n", oldCfg.HasProjects, newCfg.HasProjects))
		changed = true
	}
	if oldCfg.HomepageURL != newCfg.HomepageURL {
		sb.WriteString(fmt.Sprintf("*Homepage URL:* %s -> %s\n", oldCfg.HomepageURL, newCfg.HomepageURL))
		changed = true
	}
	if strings.Join(oldCfg.Topics, ",") != strings.Join(newCfg.Topics, ",") {
		sb.WriteString(fmt.Sprintf("*Topics:* %s -> %s\n", strings.Join(oldCfg.Topics, ", "), strings.Join(newCfg.Topics, ", ")))
		changed = true
	}
	if !teamAccessEqual(oldCfg.TeamAccess, newCfg.TeamAccess) {
		sb.WriteString(fmt.Sprintf("*Team Access:* %s -> %s\n", formatTeamAccessDisplay(oldCfg.TeamAccess), formatTeamAccessDisplay(newCfg.TeamAccess)))
		changed = true
	}
	if oldCfg.EnableBranchProtection != newCfg.EnableBranchProtection {
		sb.WriteString(fmt.Sprintf("*Branch Protection:* %v -> %v\n", oldCfg.EnableBranchProtection, newCfg.EnableBranchProtection))
		changed = true
	} else if newCfg.EnableBranchProtection {
		if oldCfg.RequiredReviews != newCfg.RequiredReviews {
			sb.WriteString(fmt.Sprintf("*Required Reviews:* %d -> %d\n", oldCfg.RequiredReviews, newCfg.RequiredReviews))
			changed = true
		}
		if oldCfg.DismissStaleReviews != newCfg.DismissStaleReviews {
			sb.WriteString(fmt.Sprintf("*Dismiss Stale Reviews:* %v -> %v\n", oldCfg.DismissStaleReviews, newCfg.DismissStaleReviews))
			changed = true
		}
		if oldCfg.RequireLinearHistory != newCfg.RequireLinearHistory {
			sb.WriteString(fmt.Sprintf("*Require Linear History:* %v -> %v\n", oldCfg.RequireLinearHistory, newCfg.RequireLinearHistory))
			changed = true
		}
		if oldCfg.RequireConversationResolution != newCfg.RequireConversationResolution {
			sb.WriteString(fmt.Sprintf("*Require Conversation Resolution:* %v -> %v\n", oldCfg.RequireConversationResolution, newCfg.RequireConversationResolution))
			changed = true
		}
	}
	if !changed {
		sb.WriteString("_No changes detected._\n")
	}

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false),
		nil, nil,
	)
	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	settingsActions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, settingsActions}
}

func teamAccessEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func formatTeamAccessDisplay(teams map[string]string) string {
	if len(teams) == 0 {
		return "(none)"
	}
	var keys []string
	for k := range teams {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, teams[k]))
	}
	return strings.Join(parts, ", ")
}

// --- DNS modals and confirmation blocks ---

var dnsTypeOptions = []*slack.OptionBlockObject{
	slack.NewOptionBlockObject("A", slack.NewTextBlockObject("plain_text", "A", false, false), nil),
	slack.NewOptionBlockObject("AAAA", slack.NewTextBlockObject("plain_text", "AAAA", false, false), nil),
	slack.NewOptionBlockObject("CNAME", slack.NewTextBlockObject("plain_text", "CNAME", false, false), nil),
	slack.NewOptionBlockObject("MX", slack.NewTextBlockObject("plain_text", "MX", false, false), nil),
	slack.NewOptionBlockObject("TXT", slack.NewTextBlockObject("plain_text", "TXT", false, false), nil),
}

func DnsAddModal(zone string) slack.ModalViewRequest {
	zoneCtx := slack.NewContextBlock("dns_zone",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Zone:* `%s`", zone), false, false))

	typeElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select type", false, false),
		ElemDnsType, dnsTypeOptions...)
	typeBlock := slack.NewInputBlock(BlockDnsType,
		slack.NewTextBlockObject("plain_text", "Record Type", false, false), nil, typeElem)

	nameElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "subdomain or @", false, false), ElemDnsName)
	nameBlock := slack.NewInputBlock(BlockDnsName,
		slack.NewTextBlockObject("plain_text", "Name", false, false), nil, nameElem)

	contentElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "1.2.3.4 or target.example.com", false, false), ElemDnsContent)
	contentBlock := slack.NewInputBlock(BlockDnsContent,
		slack.NewTextBlockObject("plain_text", "Content", false, false), nil, contentElem)

	proxiedOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Proxy through Cloudflare", false, false), nil)
	proxiedElem := slack.NewCheckboxGroupsBlockElement(ElemDnsProxied, proxiedOpt)
	proxiedBlock := slack.NewInputBlock(BlockDnsProxied,
		slack.NewTextBlockObject("plain_text", "Proxied?", false, false),
		slack.NewTextBlockObject("plain_text", "Only applies to A, AAAA, and CNAME records", false, false), proxiedElem)
	proxiedBlock.Optional = true

	priorityElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "10", false, false), ElemDnsPriority)
	priorityBlock := slack.NewInputBlock(BlockDnsPriority,
		slack.NewTextBlockObject("plain_text", "Priority", false, false),
		slack.NewTextBlockObject("plain_text", "Required for MX records", false, false), priorityElem)
	priorityBlock.Optional = true

	commentElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "What is this record for?", false, false), ElemDnsComment)
	commentBlock := slack.NewInputBlock(BlockDnsComment,
		slack.NewTextBlockObject("plain_text", "Comment", false, false), nil, commentElem)
	commentBlock.Optional = true

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why is this DNS record needed?", false, false), ElemJustification)
	justElem.Multiline = true
	justElem.MinLength = 20
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Add DNS Record", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackDnsAdd,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				zoneCtx, typeBlock, nameBlock, contentBlock,
				proxiedBlock, priorityBlock, commentBlock, justBlock,
			},
		},
	}
}

func DnsRemoveModal(zone string, records []DnsRecordOption) slack.ModalViewRequest {
	zoneCtx := slack.NewContextBlock("dns_zone",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Zone:* `%s`", zone), false, false))

	recOpts := make([]*slack.OptionBlockObject, len(records))
	for i, r := range records {
		recOpts[i] = slack.NewOptionBlockObject(r.Key,
			slack.NewTextBlockObject("plain_text", r.Label, false, false), nil)
	}
	recElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a record...", false, false),
		ElemDnsRecord, recOpts...)
	recBlock := slack.NewInputBlock(BlockDnsRecord,
		slack.NewTextBlockObject("plain_text", "Record to Remove", false, false), nil, recElem)

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why should this record be removed?", false, false), ElemJustification)
	justElem.Multiline = true
	justElem.MinLength = 20
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Remove DNS Record", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackDnsRemove,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{zoneCtx, recBlock, justBlock},
		},
	}
}

func DnsSelectRecordModal(zone string, records []DnsRecordOption) slack.ModalViewRequest {
	zoneCtx := slack.NewContextBlock("dns_zone",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Zone:* `%s`", zone), false, false))

	recOpts := make([]*slack.OptionBlockObject, len(records))
	for i, r := range records {
		recOpts[i] = slack.NewOptionBlockObject(r.Key,
			slack.NewTextBlockObject("plain_text", r.Label, false, false), nil)
	}
	recElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select a record...", false, false),
		ElemDnsRecord, recOpts...)
	recBlock := slack.NewInputBlock(BlockDnsRecord,
		slack.NewTextBlockObject("plain_text", "DNS Record", false, false), nil, recElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Select DNS Record", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Next", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackDnsSelectRecord,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{zoneCtx, recBlock},
		},
	}
}

func DnsUpdateModal(zone string, cfg conversation.DnsConfig) slack.ModalViewRequest {
	zoneCtx := slack.NewContextBlock("dns_zone",
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Zone:* `%s` | *Record:* `%s`", zone, cfg.RecordKey), false, false))

	typeElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select type", false, false),
		ElemDnsType, dnsTypeOptions...)
	for _, o := range dnsTypeOptions {
		if o.Value == cfg.Type {
			typeElem.InitialOption = o
			break
		}
	}
	typeBlock := slack.NewInputBlock(BlockDnsType,
		slack.NewTextBlockObject("plain_text", "Record Type", false, false), nil, typeElem)

	nameElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "subdomain or @", false, false), ElemDnsName)
	nameElem.InitialValue = cfg.Name
	nameBlock := slack.NewInputBlock(BlockDnsName,
		slack.NewTextBlockObject("plain_text", "Name", false, false), nil, nameElem)

	contentElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "1.2.3.4 or target.example.com", false, false), ElemDnsContent)
	contentElem.InitialValue = cfg.Content
	contentBlock := slack.NewInputBlock(BlockDnsContent,
		slack.NewTextBlockObject("plain_text", "Content", false, false), nil, contentElem)

	proxiedOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Proxy through Cloudflare", false, false), nil)
	proxiedElem := slack.NewCheckboxGroupsBlockElement(ElemDnsProxied, proxiedOpt)
	if cfg.Proxied {
		proxiedElem.InitialOptions = []*slack.OptionBlockObject{proxiedOpt}
	}
	proxiedBlock := slack.NewInputBlock(BlockDnsProxied,
		slack.NewTextBlockObject("plain_text", "Proxied?", false, false),
		slack.NewTextBlockObject("plain_text", "Only applies to A, AAAA, and CNAME records", false, false), proxiedElem)
	proxiedBlock.Optional = true

	priorityElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "10", false, false), ElemDnsPriority)
	if cfg.Priority > 0 {
		priorityElem.InitialValue = strconv.Itoa(cfg.Priority)
	}
	priorityBlock := slack.NewInputBlock(BlockDnsPriority,
		slack.NewTextBlockObject("plain_text", "Priority", false, false),
		slack.NewTextBlockObject("plain_text", "Required for MX records", false, false), priorityElem)
	priorityBlock.Optional = true

	commentElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "What is this record for?", false, false), ElemDnsComment)
	if cfg.Comment != "" {
		commentElem.InitialValue = cfg.Comment
	}
	commentBlock := slack.NewInputBlock(BlockDnsComment,
		slack.NewTextBlockObject("plain_text", "Comment", false, false), nil, commentElem)
	commentBlock.Optional = true

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why are these changes needed?", false, false), ElemJustification)
	justElem.Multiline = true
	justElem.MinLength = 20
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Update DNS Record", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackDnsUpdate,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				zoneCtx, typeBlock, nameBlock, contentBlock,
				proxiedBlock, priorityBlock, commentBlock, justBlock,
			},
		},
	}
}

func DnsAddConfirmationBlocks(zone string, cfg conversation.DnsConfig, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*You're about to add a DNS record. Please confirm.*\n\n")
	sb.WriteString(fmt.Sprintf("*Zone:* `%s`\n", zone))
	sb.WriteString(fmt.Sprintf("*Type:* %s\n", cfg.Type))
	sb.WriteString(fmt.Sprintf("*Name:* %s\n", cfg.Name))
	sb.WriteString(fmt.Sprintf("*Content:* `%s`\n", cfg.Content))
	if proxied, _ := dnsFieldRelevant(cfg.Type, "proxied"); proxied {
		sb.WriteString(fmt.Sprintf("*Proxied:* %v\n", cfg.Proxied))
	}
	if priority, _ := dnsFieldRelevant(cfg.Type, "priority"); priority {
		sb.WriteString(fmt.Sprintf("*Priority:* %d\n", cfg.Priority))
	}
	if cfg.Comment != "" {
		sb.WriteString(fmt.Sprintf("*Comment:* %s\n", cfg.Comment))
	}
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n", justification))

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false), nil, nil)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	actions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, actions}
}

func DnsRemoveConfirmationBlocks(zone, recordKey, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*You're about to remove a DNS record. Please confirm.*\n\n")
	sb.WriteString(fmt.Sprintf("*Zone:* `%s`\n", zone))
	sb.WriteString(fmt.Sprintf("*Record:* `%s`\n", recordKey))
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n", justification))

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false), nil, nil)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	actions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, actions}
}

func DnsUpdateConfirmationBlocks(zone string, oldCfg, newCfg conversation.DnsConfig, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*Here are the proposed DNS record changes. Please review before submitting.*\n\n")
	sb.WriteString(fmt.Sprintf("*Zone:* `%s`\n", zone))
	sb.WriteString(fmt.Sprintf("*Record:* `%s`\n", oldCfg.RecordKey))
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n\n", justification))

	changed := false
	if oldCfg.Type != newCfg.Type {
		sb.WriteString(fmt.Sprintf("*Type:* %s -> %s\n", oldCfg.Type, newCfg.Type))
		changed = true
	}
	if oldCfg.Name != newCfg.Name {
		sb.WriteString(fmt.Sprintf("*Name:* %s -> %s\n", oldCfg.Name, newCfg.Name))
		changed = true
	}
	if oldCfg.Content != newCfg.Content {
		sb.WriteString(fmt.Sprintf("*Content:* `%s` -> `%s`\n", oldCfg.Content, newCfg.Content))
		changed = true
	}
	if oldCfg.Proxied != newCfg.Proxied {
		sb.WriteString(fmt.Sprintf("*Proxied:* %v -> %v\n", oldCfg.Proxied, newCfg.Proxied))
		changed = true
	}
	if oldCfg.Priority != newCfg.Priority {
		sb.WriteString(fmt.Sprintf("*Priority:* %d -> %d\n", oldCfg.Priority, newCfg.Priority))
		changed = true
	}
	if oldCfg.Comment != newCfg.Comment {
		sb.WriteString(fmt.Sprintf("*Comment:* %s -> %s\n", oldCfg.Comment, newCfg.Comment))
		changed = true
	}
	if !changed {
		sb.WriteString("_No changes detected._\n")
	}

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false), nil, nil)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	actions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, actions}
}

// dnsFieldRelevant checks if a field is relevant for a DNS record type.
func dnsFieldRelevant(typ, field string) (bool, bool) {
	switch field {
	case "proxied":
		return typ == "A" || typ == "AAAA" || typ == "CNAME", false
	case "priority":
		return typ == "MX", false
	default:
		return false, false
	}
}

// --- Org Settings ---

var orgPermissionOptions = []*slack.OptionBlockObject{
	slack.NewOptionBlockObject("read", slack.NewTextBlockObject("plain_text", "Read", false, false), nil),
	slack.NewOptionBlockObject("write", slack.NewTextBlockObject("plain_text", "Write", false, false), nil),
	slack.NewOptionBlockObject("admin", slack.NewTextBlockObject("plain_text", "Admin", false, false), nil),
	slack.NewOptionBlockObject("none", slack.NewTextBlockObject("plain_text", "None", false, false), nil),
}

func OrgSettingsModal(cfg conversation.OrgConfig) slack.ModalViewRequest {
	nameElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Organization name", false, false), ElemOrgName)
	nameElem.InitialValue = cfg.Name
	nameBlock := slack.NewInputBlock(BlockOrgName,
		slack.NewTextBlockObject("plain_text", "Name", false, false), nil, nameElem)

	billingElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "billing@example.com", false, false), ElemOrgBilling)
	billingElem.InitialValue = cfg.BillingEmail
	billingBlock := slack.NewInputBlock(BlockOrgBilling,
		slack.NewTextBlockObject("plain_text", "Billing Email", false, false), nil, billingElem)

	blogElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "https://example.com", false, false), ElemOrgBlog)
	blogElem.InitialValue = cfg.Blog
	blogBlock := slack.NewInputBlock(BlockOrgBlog,
		slack.NewTextBlockObject("plain_text", "Blog", false, false), nil, blogElem)
	blogBlock.Optional = true

	descElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Organization description", false, false), ElemOrgDesc)
	descElem.InitialValue = cfg.Description
	descBlock := slack.NewInputBlock(BlockOrgDesc,
		slack.NewTextBlockObject("plain_text", "Description", false, false), nil, descElem)

	locationElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "City, Country", false, false), ElemOrgLocation)
	locationElem.InitialValue = cfg.Location
	locationBlock := slack.NewInputBlock(BlockOrgLocation,
		slack.NewTextBlockObject("plain_text", "Location", false, false), nil, locationElem)
	locationBlock.Optional = true

	permElem := slack.NewOptionsSelectBlockElement("static_select",
		slack.NewTextBlockObject("plain_text", "Select permission", false, false),
		ElemOrgPermission, orgPermissionOptions...)
	for _, o := range orgPermissionOptions {
		if o.Value == cfg.DefaultRepoPermission {
			permElem.InitialOption = o
			break
		}
	}
	permBlock := slack.NewInputBlock(BlockOrgPermission,
		slack.NewTextBlockObject("plain_text", "Default Repository Permission", false, false), nil, permElem)

	membersOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Allow members to create repositories", false, false), nil)
	membersElem := slack.NewCheckboxGroupsBlockElement(ElemOrgMembersCreate, membersOpt)
	if cfg.MembersCanCreateRepos {
		membersElem.InitialOptions = []*slack.OptionBlockObject{membersOpt}
	}
	membersBlock := slack.NewInputBlock(BlockOrgMembersCreate,
		slack.NewTextBlockObject("plain_text", "Members Can Create Repos", false, false), nil, membersElem)
	membersBlock.Optional = true

	signoffOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Require sign-off on web commits", false, false), nil)
	signoffElem := slack.NewCheckboxGroupsBlockElement(ElemOrgSignoff, signoffOpt)
	if cfg.WebCommitSignoffRequired {
		signoffElem.InitialOptions = []*slack.OptionBlockObject{signoffOpt}
	}
	signoffBlock := slack.NewInputBlock(BlockOrgSignoff,
		slack.NewTextBlockObject("plain_text", "Web Commit Sign-off", false, false), nil, signoffElem)
	signoffBlock.Optional = true

	depAlertsOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable Dependabot alerts for new repos", false, false), nil)
	depAlertsElem := slack.NewCheckboxGroupsBlockElement(ElemOrgDepAlerts, depAlertsOpt)
	if cfg.DependabotAlerts {
		depAlertsElem.InitialOptions = []*slack.OptionBlockObject{depAlertsOpt}
	}
	depAlertsBlock := slack.NewInputBlock(BlockOrgDepAlerts,
		slack.NewTextBlockObject("plain_text", "Dependabot Alerts", false, false), nil, depAlertsElem)
	depAlertsBlock.Optional = true

	depSecOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable Dependabot security updates for new repos", false, false), nil)
	depSecElem := slack.NewCheckboxGroupsBlockElement(ElemOrgDepSec, depSecOpt)
	if cfg.DependabotSecurityUpdates {
		depSecElem.InitialOptions = []*slack.OptionBlockObject{depSecOpt}
	}
	depSecBlock := slack.NewInputBlock(BlockOrgDepSec,
		slack.NewTextBlockObject("plain_text", "Dependabot Security Updates", false, false), nil, depSecElem)
	depSecBlock.Optional = true

	depGraphOpt := slack.NewOptionBlockObject("true",
		slack.NewTextBlockObject("plain_text", "Enable dependency graph for new repos", false, false), nil)
	depGraphElem := slack.NewCheckboxGroupsBlockElement(ElemOrgDepGraph, depGraphOpt)
	if cfg.DependencyGraph {
		depGraphElem.InitialOptions = []*slack.OptionBlockObject{depGraphOpt}
	}
	depGraphBlock := slack.NewInputBlock(BlockOrgDepGraph,
		slack.NewTextBlockObject("plain_text", "Dependency Graph", false, false), nil, depGraphElem)
	depGraphBlock.Optional = true

	justElem := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject("plain_text", "Why are these changes needed?", false, false), ElemJustification)
	justElem.Multiline = true
	justElem.MinLength = 20
	justBlock := slack.NewInputBlock(BlockJustification,
		slack.NewTextBlockObject("plain_text", "Justification", false, false),
		slack.NewTextBlockObject("plain_text", "Minimum 20 characters. This will appear in the PR description.", false, false), justElem)

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		Title:      slack.NewTextBlockObject("plain_text", "Org Settings", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Review", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		CallbackID: CallbackOrgSettings,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				nameBlock, billingBlock, blogBlock, descBlock, locationBlock,
				permBlock, membersBlock, signoffBlock,
				depAlertsBlock, depSecBlock, depGraphBlock,
				justBlock,
			},
		},
	}
}

func OrgSettingsConfirmationBlocks(oldCfg, newCfg conversation.OrgConfig, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*Here are the proposed org setting changes. Please review before submitting.*\n\n")
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n\n", justification))

	changed := false
	if oldCfg.Name != newCfg.Name {
		sb.WriteString(fmt.Sprintf("*Name:* %s -> %s\n", oldCfg.Name, newCfg.Name))
		changed = true
	}
	if oldCfg.BillingEmail != newCfg.BillingEmail {
		sb.WriteString(fmt.Sprintf("*Billing Email:* %s -> %s\n", oldCfg.BillingEmail, newCfg.BillingEmail))
		changed = true
	}
	if oldCfg.Blog != newCfg.Blog {
		sb.WriteString(fmt.Sprintf("*Blog:* %s -> %s\n", oldCfg.Blog, newCfg.Blog))
		changed = true
	}
	if oldCfg.Description != newCfg.Description {
		sb.WriteString(fmt.Sprintf("*Description:* %s -> %s\n", oldCfg.Description, newCfg.Description))
		changed = true
	}
	if oldCfg.Location != newCfg.Location {
		sb.WriteString(fmt.Sprintf("*Location:* %s -> %s\n", oldCfg.Location, newCfg.Location))
		changed = true
	}
	if oldCfg.DefaultRepoPermission != newCfg.DefaultRepoPermission {
		sb.WriteString(fmt.Sprintf("*Default Repo Permission:* %s -> %s\n", oldCfg.DefaultRepoPermission, newCfg.DefaultRepoPermission))
		changed = true
	}
	if oldCfg.MembersCanCreateRepos != newCfg.MembersCanCreateRepos {
		sb.WriteString(fmt.Sprintf("*Members Can Create Repos:* %v -> %v\n", oldCfg.MembersCanCreateRepos, newCfg.MembersCanCreateRepos))
		changed = true
	}
	if oldCfg.WebCommitSignoffRequired != newCfg.WebCommitSignoffRequired {
		sb.WriteString(fmt.Sprintf("*Web Commit Sign-off:* %v -> %v\n", oldCfg.WebCommitSignoffRequired, newCfg.WebCommitSignoffRequired))
		changed = true
	}
	if oldCfg.DependabotAlerts != newCfg.DependabotAlerts {
		sb.WriteString(fmt.Sprintf("*Dependabot Alerts:* %v -> %v\n", oldCfg.DependabotAlerts, newCfg.DependabotAlerts))
		changed = true
	}
	if oldCfg.DependabotSecurityUpdates != newCfg.DependabotSecurityUpdates {
		sb.WriteString(fmt.Sprintf("*Dependabot Security Updates:* %v -> %v\n", oldCfg.DependabotSecurityUpdates, newCfg.DependabotSecurityUpdates))
		changed = true
	}
	if oldCfg.DependencyGraph != newCfg.DependencyGraph {
		sb.WriteString(fmt.Sprintf("*Dependency Graph:* %v -> %v\n", oldCfg.DependencyGraph, newCfg.DependencyGraph))
		changed = true
	}
	if !changed {
		sb.WriteString("_No changes detected._\n")
	}

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false), nil, nil)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	actions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, actions}
}

func ConfirmationBlocks(name, description, visibility string, topics []string, teamAccess map[string]string, defaultBranch string, hasIssues bool, enableProtection bool, dismissStale, requireLinear, requireConvRes bool, requiredReviews int, autoMerge, updateBranch, deleteBranch, hasDiscussions, hasProjects bool, homepageURL, justification string) []slack.Block {
	var sb strings.Builder
	sb.WriteString("*Here's a summary of the new repository. Please review before submitting.*\n\n")
	sb.WriteString(fmt.Sprintf("*Name:* `%s`\n", name))
	sb.WriteString(fmt.Sprintf("*Description:* %s\n", description))
	sb.WriteString(fmt.Sprintf("*Visibility:* %s\n", visibility))
	sb.WriteString(fmt.Sprintf("*Justification:* %s\n", justification))
	if len(topics) > 0 {
		sb.WriteString(fmt.Sprintf("*Topics:* %s\n", strings.Join(topics, ", ")))
	}
	if len(teamAccess) > 0 {
		pairs := make([]string, 0, len(teamAccess))
		for k, v := range teamAccess {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
		sb.WriteString(fmt.Sprintf("*Team Access:* %s\n", strings.Join(pairs, ", ")))
	}
	sb.WriteString(fmt.Sprintf("*Default Branch:* %s\n", defaultBranch))
	if enableProtection {
		sb.WriteString(fmt.Sprintf("*Branch Protection:* enabled (reviews=%d, dismiss_stale=%v, linear=%v, conv_res=%v)\n",
			requiredReviews, dismissStale, requireLinear, requireConvRes))
	}
	if autoMerge {
		sb.WriteString("*Auto Merge:* enabled\n")
	}
	if updateBranch {
		sb.WriteString("*Update Branch:* enabled\n")
	}
	if deleteBranch {
		sb.WriteString("*Delete Branch on Merge:* enabled\n")
	}

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", sb.String(), false, false),
		nil, nil,
	)

	confirmBtn := slack.NewButtonBlockElement(ActionConfirm, "confirm",
		slack.NewTextBlockObject("plain_text", "Confirm", false, false))
	confirmBtn.Style = "primary"
	actions := slack.NewActionBlock("confirm_actions", confirmBtn)

	return []slack.Block{section, actions}
}
