package slack

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/jae-labs/opsy/internal/conversation"
	ghclient "github.com/jae-labs/opsy/internal/github"
	hcleditor "github.com/jae-labs/opsy/internal/hcl"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

const (
	pathGitHubRepos   = "iac/terraform/github/locals_repos.tf"
	pathGitHubMembers = "iac/terraform/github/locals_members.tf"
	pathGitHubOrg     = "iac/terraform/github/locals_org.tf"
	pathCloudflareDNS = "iac/terraform/cloudflare/locals_dns.tf"
)

// adminUserIDs is the set of Slack user IDs allowed to approve PRs via reaction.
var adminUserIDs = map[string]bool{
	"UF7BMAWCT": true,
}

var prURLPattern = regexp.MustCompile(`/pull/(\d+)`)

type Handler struct {
	api    *slack.Client
	sm     *socketmode.Client
	store  *conversation.Store
	gh     *ghclient.Client
	logger *slog.Logger

	// prMessages maps "channelID:messageTS" -> PR URL for reaction-based approval.
	// Entries are removed after approval to prevent duplicates.
	prMessages sync.Map
}

func NewHandler(api *slack.Client, sm *socketmode.Client, gh *ghclient.Client, logger *slog.Logger) *Handler {
	return &Handler{
		api:    api,
		sm:     sm,
		store:  conversation.NewStore(),
		gh:     gh,
		logger: logger,
	}
}

func (h *Handler) Run() {
	go h.eventLoop()
	h.sm.Run()
}

// reply sends a message in the flow's thread.
func (h *Handler) reply(state *conversation.State, opts ...slack.MsgOption) {
	opts = append(opts, slack.MsgOptionTS(state.ThreadTS))
	_, _, _ = h.api.PostMessage(state.ChannelID, opts...)
}

// replyPR sends the PR-created message and tracks it for reaction-based approval.
func (h *Handler) replyPR(state *conversation.State, prURL string) {
	opts := []slack.MsgOption{
		slack.MsgOptionText(fmt.Sprintf("PR created. %s\nReact with :thumbsup: to approve.", prURL), false),
		slack.MsgOptionTS(state.ThreadTS),
	}
	_, msgTS, _ := h.api.PostMessage(state.ChannelID, opts...)
	if msgTS != "" {
		key := state.ChannelID + ":" + msgTS
		h.prMessages.Store(key, prURL)
	}
}

// updateMessage replaces the blocks of an existing message (used to lock dropdowns).
func (h *Handler) updateMessage(state *conversation.State, messageTS string, opts ...slack.MsgOption) {
	opts = append(opts, slack.MsgOptionTS(messageTS))
	_, _, _, _ = h.api.UpdateMessage(state.ChannelID, messageTS, opts...)
}

func (h *Handler) eventLoop() {
	for evt := range h.sm.Events {
		switch evt.Type {
		case socketmode.EventTypeEventsAPI:
			h.handleEventsAPI(evt)
		case socketmode.EventTypeInteractive:
			h.handleInteractive(evt)
		default:
			h.logger.Debug("unhandled event type", "type", evt.Type)
		}
	}
}

func (h *Handler) handleEventsAPI(evt socketmode.Event) {
	eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}
	h.sm.Ack(*evt.Request)

	switch ev := eventsAPI.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		if ev.ChannelType != "im" || ev.BotID != "" || ev.SubType != "" {
			return
		}

		if ev.ThreadTimeStamp != "" {
			// message inside an existing thread
			h.handleThreadReply(ev.User, ev.Channel, ev.Text, ev.ThreadTimeStamp)
		} else {
			// top-level message -> new flow
			h.handleNewFlow(ev.User, ev.Channel, ev.TimeStamp)
		}
	case *slackevents.ReactionAddedEvent:
		if ev.Reaction == "+1" || ev.Reaction == "thumbsup" {
			h.handlePRApproval(ev.User, ev.Item.Channel, ev.Item.Timestamp)
		}
	}
}

// handleNewFlow starts a fresh flow threaded to the user's message.
func (h *Handler) handleNewFlow(userID, channelID, messageTS string) {
	state := h.store.Create(messageTS, channelID, userID)

	_, _, err := h.api.PostMessage(channelID,
		slack.MsgOptionBlocks(WelcomeBlocks()...),
		slack.MsgOptionTS(messageTS),
	)
	if err != nil {
		h.logger.Error("failed to send welcome", "error", err, "user", userID)
		return
	}
	_ = state
}

// handleThreadReply handles text typed inside a flow's thread (e.g. "cancel").
func (h *Handler) handleThreadReply(userID, channelID, text, threadTS string) {
	state := h.store.Get(threadTS)
	if state == nil {
		_, _, _ = h.api.PostMessage(channelID,
			slack.MsgOptionText("This flow has already ended. Send a new message to start another.", false),
			slack.MsgOptionTS(threadTS),
		)
		return
	}

	if strings.EqualFold(strings.TrimSpace(text), "cancel") {
		h.reply(state, slack.MsgOptionText("Cancelled. Send a new message when you're ready.", false))
		h.store.Delete(threadTS)
		return
	}

	h.reply(state, slack.MsgOptionText("Still in progress. Complete the steps above, or type 'cancel' to start over.", false))
}

func (h *Handler) handleInteractive(evt socketmode.Event) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		return
	}

	switch callback.Type {
	case slack.InteractionTypeBlockActions:
		h.sm.Ack(*evt.Request)
		h.handleBlockAction(callback)
	case slack.InteractionTypeViewSubmission:
		h.handleViewSubmission(evt, callback)
	default:
		h.sm.Ack(*evt.Request)
	}
}

func (h *Handler) handleBlockAction(callback slack.InteractionCallback) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		return
	}
	action := callback.ActionCallback.BlockActions[0]

	// resolve flow from thread
	threadTS := callback.Message.ThreadTimestamp
	if threadTS == "" {
		threadTS = callback.Message.Timestamp
	}
	state := h.store.Get(threadTS)
	if state == nil {
		h.logger.Warn("no flow for thread", "thread_ts", threadTS)
		return
	}

	messageTS := callback.Message.Timestamp

	switch action.ActionID {
	case ActionCategorySelect:
		h.handleCategorySelect(state, action.SelectedOption.Value, messageTS)
	case ActionResourceSelect:
		h.handleResourceSelect(state, action.SelectedOption.Value, callback.TriggerID, messageTS)
	case ActionActionSelect:
		h.handleActionSelect(state, action.SelectedOption.Value, callback.TriggerID, messageTS)
	case ActionConfirm:
		h.handleConfirm(state, messageTS)
	case ActionCancel:
		h.reply(state, slack.MsgOptionText("Cancelled. Send a new message when you're ready.", false))
		h.store.Delete(state.ThreadTS)
	}
}

func (h *Handler) handleCategorySelect(state *conversation.State, category, messageTS string) {
	state.Category = category
	state.Phase = conversation.PhaseCategorySelected

	blocks := ResourceBlocks(category)
	h.reply(state, slack.MsgOptionBlocks(blocks...))

	label := labelForValue(categories, category)
	h.updateMessage(state, messageTS, slack.MsgOptionBlocks(LockedCategoryBlocks(label)...))

	// if no resources defined for this category, ResourceBlocks returns ComingSoon and flow ends
	if _, ok := resourceOptions[category]; !ok {
		h.store.Delete(state.ThreadTS)
	}
}

func (h *Handler) handleResourceSelect(state *conversation.State, resource, triggerID, messageTS string) {
	state.ResourceType = resource
	state.Phase = conversation.PhaseResourceSelected

	switch resource {
	case "repo", "dns":
		blocks := ActionBlocks(resource)
		h.reply(state, slack.MsgOptionBlocks(blocks...))
	case "org_settings":
		h.handleOrgSettingsResource(state, triggerID)
	default:
		h.reply(state, slack.MsgOptionBlocks(ComingSoonBlocks(resource)...))
		h.store.Delete(state.ThreadTS)
	}

	label := labelForValue(resourceOptions[state.Category], resource)
	h.updateMessage(state, messageTS, slack.MsgOptionBlocks(LockedResourceBlocks(state.Category, label)...))
}

func (h *Handler) handleActionSelect(state *conversation.State, actionType, triggerID, messageTS string) {
	state.ActionType = actionType
	state.Phase = conversation.PhaseActionSelected

	switch state.ResourceType {
	case "dns":
		h.handleDnsAction(state, actionType, triggerID)
	default:
		h.handleRepoAction(state, actionType, triggerID)
	}

	label := labelForValue(actionOptions[state.ResourceType], actionType)
	h.updateMessage(state, messageTS, slack.MsgOptionBlocks(LockedActionBlocks(label)...))
}

func (h *Handler) handleRepoAction(state *conversation.State, actionType, triggerID string) {
	switch actionType {
	case "add":
		modal := RepoStep1Modal()
		modal.PrivateMetadata = state.ThreadTS
		_, err := h.api.OpenView(triggerID, modal)
		if err != nil {
			h.logger.Error("failed to open modal", "error", err, "user", state.UserID)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	case "delete":
		repos := h.fetchRepoNames()
		if len(repos) == 0 {
			h.reply(state, slack.MsgOptionText("No repositories found. There's nothing to remove.", false))
			return
		}
		modal := DeleteRepoModal(repos)
		modal.PrivateMetadata = state.ThreadTS
		_, err := h.api.OpenView(triggerID, modal)
		if err != nil {
			h.logger.Error("failed to open delete modal", "error", err, "user", state.UserID)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	case "settings":
		repos := h.fetchRepoNames()
		if len(repos) == 0 {
			h.reply(state, slack.MsgOptionText("No repositories found. There's nothing to update.", false))
			return
		}
		modal := SelectRepoModal(repos)
		modal.PrivateMetadata = state.ThreadTS
		_, err := h.api.OpenView(triggerID, modal)
		if err != nil {
			h.logger.Error("failed to open select repo modal", "error", err, "user", state.UserID)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	}
}

func (h *Handler) handleDnsAction(state *conversation.State, actionType, triggerID string) {
	src, err := h.fetchCloudflareHCL()
	if err != nil {
		h.reply(state, slack.MsgOptionText("Failed to fetch Cloudflare configuration.", false))
		h.store.Delete(state.ThreadTS)
		return
	}

	zones, err := hcleditor.ExistingZones(src)
	if err != nil || len(zones) == 0 {
		h.reply(state, slack.MsgOptionText("No DNS zones found.", false))
		h.store.Delete(state.ThreadTS)
		return
	}

	// auto-select when only one zone exists
	zone := zones[0]
	state.TargetZone = zone

	switch actionType {
	case "add":
		modal := DnsAddModal(zone)
		modal.PrivateMetadata = state.ThreadTS
		if _, err := h.api.OpenView(triggerID, modal); err != nil {
			h.logger.Error("failed to open dns add modal", "error", err)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	case "delete":
		records := h.fetchDnsRecordOptions(src, zone)
		if len(records) == 0 {
			h.reply(state, slack.MsgOptionText("No DNS records found. There's nothing to remove.", false))
			return
		}
		modal := DnsRemoveModal(zone, records)
		modal.PrivateMetadata = state.ThreadTS
		if _, err := h.api.OpenView(triggerID, modal); err != nil {
			h.logger.Error("failed to open dns remove modal", "error", err)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	case "settings":
		records := h.fetchDnsRecordOptions(src, zone)
		if len(records) == 0 {
			h.reply(state, slack.MsgOptionText("No DNS records found. There's nothing to update.", false))
			return
		}
		modal := DnsSelectRecordModal(zone, records)
		modal.PrivateMetadata = state.ThreadTS
		if _, err := h.api.OpenView(triggerID, modal); err != nil {
			h.logger.Error("failed to open dns select record modal", "error", err)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
		}
	}
}

func (h *Handler) handleOrgSettingsResource(state *conversation.State, triggerID string) {
	src, err := h.fetchOrgHCLSource()
	if err != nil {
		h.reply(state, slack.MsgOptionText("Failed to fetch GitHub configuration.", false))
		h.store.Delete(state.ThreadTS)
		return
	}

	cfg, err := hcleditor.ExtractOrgSettings(src)
	if err != nil {
		h.logger.Error("failed to extract org settings", "error", err)
		h.reply(state, slack.MsgOptionText("Failed to read org settings from configuration.", false))
		h.store.Delete(state.ThreadTS)
		return
	}
	state.OrgConfig = cfg

	modal := OrgSettingsModal(state.OrgConfig)
	modal.PrivateMetadata = state.ThreadTS
	if _, err := h.api.OpenView(triggerID, modal); err != nil {
		h.logger.Error("failed to open org settings modal", "error", err)
		h.reply(state, slack.MsgOptionText(fmt.Sprintf("Failed to open modal: %v", err), false))
	}
}

func (h *Handler) handleViewSubmission(evt socketmode.Event, callback slack.InteractionCallback) {
	// resolve flow from PrivateMetadata (threadTS)
	threadTS := callback.View.PrivateMetadata
	state := h.store.Get(threadTS)
	if state == nil {
		h.logger.Warn("no flow for modal submission", "thread_ts", threadTS)
		h.sm.Ack(*evt.Request)
		return
	}

	values := callback.View.State.Values

	h.logger.Info("view submission received",
		"callback_id", callback.View.CallbackID,
		"user", state.UserID,
		"thread_ts", threadTS,
	)

	switch callback.View.CallbackID {
	case CallbackRepoStep1:
		if errs := validateRepoStep1(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		repoName := values[BlockName][ElemName].Value
		if msg := h.checkRepoAlreadyExists(repoName); msg != "" {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          map[string]string{BlockName: msg},
			})
			return
		}
		state.RepoConfig.Name = repoName
		state.RepoConfig.Description = values[BlockDescription][ElemDescription].Value
		state.RepoConfig.Visibility = values[BlockVisibility][ElemVisibility].SelectedOption.Value
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.RepoConfig.HasIssues = true
		state.Phase = conversation.PhaseWizardStep1

		h.logger.Info("step1 parsed",
			"name", state.RepoConfig.Name,
			"description", state.RepoConfig.Description,
			"visibility", state.RepoConfig.Visibility,
		)

		state.AvailableTeams = h.fetchTeamNames()
		modal := RepoStep2Modal(state.AvailableTeams)
		modal.PrivateMetadata = threadTS

		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackRepoStep2:
		if errs := validateRepoStep2(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		if topicsVal, ok := values[BlockTopics]; ok {
			raw := topicsVal[ElemTopics].Value
			if raw != "" {
				topics := strings.Split(raw, ",")
				for i, t := range topics {
					topics[i] = strings.TrimSpace(t)
				}
				state.RepoConfig.Topics = topics
			}
		}
		state.RepoConfig.TeamAccess = parseTeamRoleValues(values, state.AvailableTeams)
		state.RepoConfig.DefaultBranch = values[BlockDefBranch][ElemDefBranch].Value
		state.Phase = conversation.PhaseWizardStep2

		h.logger.Info("step2 parsed",
			"topics", state.RepoConfig.Topics,
			"team_access", state.RepoConfig.TeamAccess,
			"default_branch", state.RepoConfig.DefaultBranch,
		)

		modal := RepoStep3Modal()
		modal.PrivateMetadata = threadTS
		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackRepoStep3:
		if errs := validateRepoStep3(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		h.parseStep3Values(state, values)
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		rc := state.RepoConfig
		blocks := ConfirmationBlocks(
			rc.Name, rc.Description, rc.Visibility, rc.Topics, rc.TeamAccess,
			rc.DefaultBranch, rc.HasIssues, rc.EnableBranchProtection,
			rc.DismissStaleReviews, rc.RequireLinearHistory, rc.RequireConversationResolution,
			rc.RequiredReviews, rc.AllowAutoMerge, rc.AllowUpdateBranch, rc.DeleteBranchOnMerge,
			rc.HasDiscussions, rc.HasProjects, rc.HomepageURL,
			state.Justification,
		)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	case CallbackDeleteRepo:
		targetRepo := values[BlockDeleteTarget][ElemDeleteTarget].SelectedOption.Value
		if msg := h.checkRepoStillExists(targetRepo); msg != "" {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          map[string]string{BlockDeleteTarget: msg},
			})
			return
		}
		state.TargetRepo = targetRepo
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		blocks := DeleteConfirmationBlocks(state.TargetRepo, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	case CallbackSelectRepo:
		repoName := values[BlockSelectRepo][ElemSelectRepo].SelectedOption.Value
		state.TargetRepo = repoName

		src, err := h.fetchHCLSource()
		if err != nil {
			h.logger.Error("failed to fetch repos HCL for settings", "error", err)
			h.sm.Ack(*evt.Request)
			h.reply(state, slack.MsgOptionText("Failed to fetch the repository configuration.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		cfg, err := hcleditor.ExtractRepoConfig(src, repoName)
		if err != nil {
			h.logger.Error("failed to extract repo config", "error", err, "repo", repoName)
			h.sm.Ack(*evt.Request)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Could not read config for %s: %v", repoName, err), false))
			h.store.Delete(state.ThreadTS)
			return
		}
		state.RepoConfig = cfg
		state.Phase = conversation.PhaseActionSelected

		modal := SettingsStep1Modal(state.RepoConfig)
		modal.PrivateMetadata = threadTS
		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackSettingsStep1:
		if errs := validateSettingsStep1(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		state.RepoConfig.Description = values[BlockDescription][ElemDescription].Value
		state.RepoConfig.Visibility = values[BlockVisibility][ElemVisibility].SelectedOption.Value
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseWizardStep1

		state.AvailableTeams = h.fetchTeamNames()
		modal := SettingsStep2Modal(state.RepoConfig, state.AvailableTeams)
		modal.PrivateMetadata = threadTS
		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackSettingsStep2:
		if errs := validateSettingsStep2(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		if topicsVal, ok := values[BlockTopics]; ok {
			raw := topicsVal[ElemTopics].Value
			if raw != "" {
				topics := strings.Split(raw, ",")
				for i, t := range topics {
					topics[i] = strings.TrimSpace(t)
				}
				state.RepoConfig.Topics = topics
			} else {
				state.RepoConfig.Topics = nil
			}
		}
		state.RepoConfig.TeamAccess = parseTeamRoleValues(values, state.AvailableTeams)
		state.RepoConfig.DefaultBranch = values[BlockDefBranch][ElemDefBranch].Value
		state.Phase = conversation.PhaseWizardStep2

		modal := SettingsStep3Modal(state.RepoConfig)
		modal.PrivateMetadata = threadTS
		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackSettingsStep3:
		if errs := validateRepoStep3(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		h.parseStep3Values(state, values)
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		// fetch fresh HCL to get old config for comparison
		src, err := h.fetchHCLSource()
		if err != nil {
			h.reply(state, slack.MsgOptionText("Failed to fetch current config for comparison.", false))
			h.store.Delete(state.ThreadTS)
			return
		}
		oldCfg, err := hcleditor.ExtractRepoConfig(src, state.TargetRepo)
		if err != nil {
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Could not extract current settings for %s.", state.TargetRepo), false))
			h.store.Delete(state.ThreadTS)
			return
		}

		// check if anything changed
		if repoConfigEqual(oldCfg, state.RepoConfig) {
			h.reply(state, slack.MsgOptionText("Nothing has changed. No PR needed.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		blocks := SettingsConfirmationBlocks(state.TargetRepo, oldCfg, state.RepoConfig, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	// --- DNS flow callbacks ---

	case CallbackDnsAdd:
		if errs := validateDnsFields(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}

		newType := values[BlockDnsType][ElemDnsType].SelectedOption.Value
		newName := values[BlockDnsName][ElemDnsName].Value

		// check for DNS conflicts against live data
		if conflict := h.checkDnsAddConflict(state.TargetZone, newName, newType); conflict != "" {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          map[string]string{BlockDnsName: conflict},
			})
			return
		}

		state.DnsConfig.Type = newType
		state.DnsConfig.Name = newName
		state.DnsConfig.Content = values[BlockDnsContent][ElemDnsContent].Value
		if proxied, ok := values[BlockDnsProxied]; ok {
			state.DnsConfig.Proxied = len(proxied[ElemDnsProxied].SelectedOptions) > 0
		}
		if priority, ok := values[BlockDnsPriority]; ok {
			if n, err := strconv.Atoi(priority[ElemDnsPriority].Value); err == nil {
				state.DnsConfig.Priority = n
			}
		}
		if comment, ok := values[BlockDnsComment]; ok {
			state.DnsConfig.Comment = comment[ElemDnsComment].Value
		}
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		blocks := DnsAddConfirmationBlocks(state.TargetZone, state.DnsConfig, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	case CallbackDnsRemove:
		recordKey := values[BlockDnsRecord][ElemDnsRecord].SelectedOption.Value

		// verify the record still exists before proceeding
		if msg := h.checkDnsRecordStillExists(state.TargetZone, recordKey); msg != "" {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          map[string]string{BlockDnsRecord: msg},
			})
			return
		}

		state.TargetRecord = recordKey
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		blocks := DnsRemoveConfirmationBlocks(state.TargetZone, state.TargetRecord, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	case CallbackDnsSelectRecord:
		recordKey := values[BlockDnsRecord][ElemDnsRecord].SelectedOption.Value
		state.TargetRecord = recordKey

		src, err := h.fetchCloudflareHCL()
		if err != nil {
			h.logger.Error("failed to fetch DNS HCL for dns update", "error", err)
			h.sm.Ack(*evt.Request)
			h.reply(state, slack.MsgOptionText("Failed to fetch DNS configuration.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		cfg, err := hcleditor.ExtractDnsConfig(src, state.TargetZone, recordKey)
		if err != nil {
			h.logger.Error("failed to extract dns config", "error", err, "record", recordKey)
			h.sm.Ack(*evt.Request)
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Could not read config for %s: %v", recordKey, err), false))
			h.store.Delete(state.ThreadTS)
			return
		}
		state.DnsConfig = cfg

		modal := DnsUpdateModal(state.TargetZone, state.DnsConfig)
		modal.PrivateMetadata = threadTS
		resp := map[string]interface{}{
			"response_action": "update",
			"view":            modal,
		}
		h.sm.Ack(*evt.Request, resp)

	case CallbackDnsUpdate:
		if errs := validateDnsFields(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}
		state.DnsConfig.Type = values[BlockDnsType][ElemDnsType].SelectedOption.Value
		state.DnsConfig.Name = values[BlockDnsName][ElemDnsName].Value
		state.DnsConfig.Content = values[BlockDnsContent][ElemDnsContent].Value
		if proxied, ok := values[BlockDnsProxied]; ok {
			state.DnsConfig.Proxied = len(proxied[ElemDnsProxied].SelectedOptions) > 0
		} else {
			state.DnsConfig.Proxied = false
		}
		if priority, ok := values[BlockDnsPriority]; ok {
			if n, err := strconv.Atoi(priority[ElemDnsPriority].Value); err == nil {
				state.DnsConfig.Priority = n
			} else {
				state.DnsConfig.Priority = 0
			}
		} else {
			state.DnsConfig.Priority = 0
		}
		if comment, ok := values[BlockDnsComment]; ok {
			state.DnsConfig.Comment = comment[ElemDnsComment].Value
		} else {
			state.DnsConfig.Comment = ""
		}
		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		// fetch fresh HCL for old config comparison
		src, err := h.fetchCloudflareHCL()
		if err != nil {
			h.reply(state, slack.MsgOptionText("Failed to fetch current DNS config for comparison.", false))
			h.store.Delete(state.ThreadTS)
			return
		}
		oldCfg, err := hcleditor.ExtractDnsConfig(src, state.TargetZone, state.TargetRecord)
		if err != nil {
			h.reply(state, slack.MsgOptionText(fmt.Sprintf("Could not extract current config for %s.", state.TargetRecord), false))
			h.store.Delete(state.ThreadTS)
			return
		}

		if dnsConfigEqual(oldCfg, state.DnsConfig) {
			h.reply(state, slack.MsgOptionText("Nothing has changed. No PR needed.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		blocks := DnsUpdateConfirmationBlocks(state.TargetZone, oldCfg, state.DnsConfig, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	// --- Org Settings flow callback ---

	case CallbackOrgSettings:
		if errs := validateOrgSettings(values); len(errs) > 0 {
			h.sm.Ack(*evt.Request, map[string]interface{}{
				"response_action": "errors",
				"errors":          errs,
			})
			return
		}

		state.OrgConfig.Name = values[BlockOrgName][ElemOrgName].Value
		state.OrgConfig.BillingEmail = values[BlockOrgBilling][ElemOrgBilling].Value
		state.OrgConfig.Blog = values[BlockOrgBlog][ElemOrgBlog].Value
		state.OrgConfig.Description = values[BlockOrgDesc][ElemOrgDesc].Value
		state.OrgConfig.Location = values[BlockOrgLocation][ElemOrgLocation].Value
		state.OrgConfig.DefaultRepoPermission = values[BlockOrgPermission][ElemOrgPermission].SelectedOption.Value

		if mc, ok := values[BlockOrgMembersCreate]; ok {
			state.OrgConfig.MembersCanCreateRepos = len(mc[ElemOrgMembersCreate].SelectedOptions) > 0
		} else {
			state.OrgConfig.MembersCanCreateRepos = false
		}
		if so, ok := values[BlockOrgSignoff]; ok {
			state.OrgConfig.WebCommitSignoffRequired = len(so[ElemOrgSignoff].SelectedOptions) > 0
		} else {
			state.OrgConfig.WebCommitSignoffRequired = false
		}
		if da, ok := values[BlockOrgDepAlerts]; ok {
			state.OrgConfig.DependabotAlerts = len(da[ElemOrgDepAlerts].SelectedOptions) > 0
		} else {
			state.OrgConfig.DependabotAlerts = false
		}
		if ds, ok := values[BlockOrgDepSec]; ok {
			state.OrgConfig.DependabotSecurityUpdates = len(ds[ElemOrgDepSec].SelectedOptions) > 0
		} else {
			state.OrgConfig.DependabotSecurityUpdates = false
		}
		if dg, ok := values[BlockOrgDepGraph]; ok {
			state.OrgConfig.DependencyGraph = len(dg[ElemOrgDepGraph].SelectedOptions) > 0
		} else {
			state.OrgConfig.DependencyGraph = false
		}

		state.Justification = values[BlockJustification][ElemJustification].Value
		state.Phase = conversation.PhaseConfirming
		h.sm.Ack(*evt.Request)

		// fetch fresh HCL for comparison
		src, err := h.fetchOrgHCLSource()
		if err != nil {
			h.reply(state, slack.MsgOptionText("Failed to fetch current config for comparison.", false))
			h.store.Delete(state.ThreadTS)
			return
		}
		oldCfg, err := hcleditor.ExtractOrgSettings(src)
		if err != nil {
			h.reply(state, slack.MsgOptionText("Could not extract current org settings.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		if orgConfigEqual(oldCfg, state.OrgConfig) {
			h.reply(state, slack.MsgOptionText("Nothing has changed. No PR needed.", false))
			h.store.Delete(state.ThreadTS)
			return
		}

		blocks := OrgSettingsConfirmationBlocks(oldCfg, state.OrgConfig, state.Justification)
		h.reply(state, slack.MsgOptionBlocks(blocks...))

	default:
		h.sm.Ack(*evt.Request)
	}
}

func (h *Handler) parseStep3Values(state *conversation.State, values map[string]map[string]slack.BlockAction) {
	rc := &state.RepoConfig

	if prot, ok := values[BlockProtection]; ok {
		rc.EnableBranchProtection = len(prot[ElemProtection].SelectedOptions) > 0
	}
	if reviews, ok := values[BlockReviews]; ok {
		if n, err := strconv.Atoi(reviews[ElemReviews].Value); err == nil {
			rc.RequiredReviews = n
		}
	}
	if dismiss, ok := values[BlockDismissStale]; ok {
		rc.DismissStaleReviews = len(dismiss[ElemDismissStale].SelectedOptions) > 0
	}
	if linear, ok := values[BlockLinear]; ok {
		rc.RequireLinearHistory = len(linear[ElemLinear].SelectedOptions) > 0
	}
	if conv, ok := values[BlockConvRes]; ok {
		rc.RequireConversationResolution = len(conv[ElemConvRes].SelectedOptions) > 0
	}
	if am, ok := values[BlockAutoMerge]; ok {
		rc.AllowAutoMerge = len(am[ElemAutoMerge].SelectedOptions) > 0
	}
	if ub, ok := values[BlockUpdateBranch]; ok {
		rc.AllowUpdateBranch = len(ub[ElemUpdateBranch].SelectedOptions) > 0
	}
	if db, ok := values[BlockDeleteBranch]; ok {
		rc.DeleteBranchOnMerge = len(db[ElemDeleteBranch].SelectedOptions) > 0
	}
	if disc, ok := values[BlockDiscussions]; ok {
		rc.HasDiscussions = len(disc[ElemDiscussions].SelectedOptions) > 0
	}
	if proj, ok := values[BlockProjects]; ok {
		rc.HasProjects = len(proj[ElemProjects].SelectedOptions) > 0
	}
	if hp, ok := values[BlockHomepage]; ok {
		rc.HomepageURL = strings.TrimSpace(hp[ElemHomepage].Value)
	}
}

func (h *Handler) handlePRApproval(userID, channelID, messageTS string) {
	if !adminUserIDs[userID] {
		return
	}

	key := channelID + ":" + messageTS
	val, loaded := h.prMessages.LoadAndDelete(key)
	if !loaded {
		return
	}
	prURL := val.(string)

	matches := prURLPattern.FindStringSubmatch(prURL)
	if len(matches) < 2 {
		h.logger.Error("could not extract PR number from URL", "url", prURL)
		return
	}
	prNumber, _ := strconv.Atoi(matches[1])

	user, err := h.api.GetUserInfo(userID)
	approverName := userID
	if err == nil {
		approverName = user.RealName
	}

	ctx := context.Background()
	body := fmt.Sprintf("Approved by %s", approverName)
	if err := h.gh.ApprovePR(ctx, prNumber, body); err != nil {
		h.logger.Error("failed to approve PR", "error", err, "pr", prNumber)
		h.api.PostMessage(channelID,
			slack.MsgOptionText(fmt.Sprintf("Failed to approve PR #%d: %s", prNumber, err), false),
			slack.MsgOptionTS(messageTS))
		return
	}

	h.api.PostMessage(channelID,
		slack.MsgOptionText(fmt.Sprintf("PR #%d approved by %s.", prNumber, approverName), false),
		slack.MsgOptionTS(messageTS))
}

func (h *Handler) fetchTeamNames() []string {
	ctx := context.Background()
	src, _, err := h.gh.GetFileContent(ctx, pathGitHubMembers)
	if err != nil {
		h.logger.Error("failed to fetch members HCL for teams", "error", err)
		return []string{"Maintainers"}
	}
	teams, err := hcleditor.ExtractTeamNames(src)
	if err != nil {
		h.logger.Error("failed to extract teams", "error", err)
		return []string{"Maintainers"}
	}
	return teams
}

func (h *Handler) handleConfirm(state *conversation.State, messageTS string) {
	if state.Phase != conversation.PhaseConfirming {
		return
	}
	state.Phase = conversation.PhaseCreatingPR

	// replace the confirm button with a status message to prevent double-clicks
	h.updateMessage(state, messageTS, slack.MsgOptionText("Creating PR... One moment.", false))

	switch state.ResourceType {
	case "dns":
		switch state.ActionType {
		case "add":
			go h.createDnsAddPR(state)
		case "delete":
			go h.createDnsRemovePR(state)
		case "settings":
			go h.createDnsUpdatePR(state)
		}
	case "org_settings":
		go h.createOrgSettingsPR(state)
	default:
		switch state.ActionType {
		case "delete":
			go h.createDeletePR(state)
		case "settings":
			go h.createSettingsPR(state)
		default:
			go h.createPR(state)
		}
	}
}

func (h *Handler) createPR(state *conversation.State) {
	ctx := context.Background()
	repo := state.RepoConfig

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathGitHubRepos)
	if err != nil {
		h.reportError(state, "fetch repos HCL", err)
		return
	}

	modified, err := hcleditor.AddRepo(src, repo)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.BranchName(repo.Name)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(github): add repository %s", repo.Name)
	if err := h.gh.UpdateFile(ctx, branch, pathGitHubRepos, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := state.UserID
	if user, err := h.api.GetUserInfo(state.UserID); err != nil {
		h.logger.Error("failed to resolve slack user name", "error", err, "user_id", state.UserID)
	} else {
		requester = user.RealName
	}

	prTitle := fmt.Sprintf("Add GitHub repository: %s", repo.Name)
	prBody := ghclient.BuildPRDescription(repo.Name, repo.Description, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func (h *Handler) createDeletePR(state *conversation.State) {
	ctx := context.Background()
	repoName := state.TargetRepo

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathGitHubRepos)
	if err != nil {
		h.reportError(state, "fetch repos HCL", err)
		return
	}

	modified, err := hcleditor.RemoveRepo(src, repoName)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.DeleteBranchName(repoName)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(github): remove repository %s", repoName)
	if err := h.gh.UpdateFile(ctx, branch, pathGitHubRepos, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := state.UserID
	if user, err := h.api.GetUserInfo(state.UserID); err != nil {
		h.logger.Error("failed to resolve slack user name", "error", err, "user_id", state.UserID)
	} else {
		requester = user.RealName
	}

	prTitle := fmt.Sprintf("Remove GitHub repository: %s", repoName)
	prBody := ghclient.BuildDeletePRDescription(repoName, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func (h *Handler) fetchRepoNames() []string {
	ctx := context.Background()
	src, _, err := h.gh.GetFileContent(ctx, pathGitHubRepos)
	if err != nil {
		h.logger.Error("failed to fetch repos HCL", "error", err)
		return nil
	}
	names, err := hcleditor.ExistingRepoNames(src)
	if err != nil {
		h.logger.Error("failed to extract repo names", "error", err)
		return nil
	}
	return names
}

// checkRepoAlreadyExists returns an error message if a repo with the given name
// already exists in the terraform config.
func (h *Handler) checkRepoAlreadyExists(name string) string {
	names := h.fetchRepoNames()
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return fmt.Sprintf("Repository %q already exists.", n)
		}
	}
	return ""
}

// checkRepoStillExists returns an error message if the repo no longer exists
// in the terraform config.
func (h *Handler) checkRepoStillExists(name string) string {
	names := h.fetchRepoNames()
	for _, n := range names {
		if n == name {
			return ""
		}
	}
	return fmt.Sprintf("Repository %q no longer exists. It may have been removed already.", name)
}

func (h *Handler) fetchHCLSource() ([]byte, error) {
	ctx := context.Background()
	src, _, err := h.gh.GetFileContent(ctx, pathGitHubRepos)
	return src, err
}

func (h *Handler) fetchOrgHCLSource() ([]byte, error) {
	ctx := context.Background()
	src, _, err := h.gh.GetFileContent(ctx, pathGitHubOrg)
	return src, err
}

func (h *Handler) createSettingsPR(state *conversation.State) {
	ctx := context.Background()
	repoName := state.TargetRepo

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathGitHubRepos)
	if err != nil {
		h.reportError(state, "fetch repos HCL", err)
		return
	}

	modified, err := hcleditor.UpdateRepo(src, repoName, state.RepoConfig)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.SettingsBranchName(repoName)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(github): update repository settings for %s", repoName)
	if err := h.gh.UpdateFile(ctx, branch, pathGitHubRepos, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := state.UserID
	if user, err := h.api.GetUserInfo(state.UserID); err != nil {
		h.logger.Error("failed to resolve slack user name", "error", err, "user_id", state.UserID)
	} else {
		requester = user.RealName
	}

	prTitle := fmt.Sprintf("Update GitHub repository settings: %s", repoName)
	prBody := ghclient.BuildSettingsPRDescription(repoName, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

// --- DNS PR creation ---

func (h *Handler) createDnsAddPR(state *conversation.State) {
	ctx := context.Background()
	zone := state.TargetZone
	cfg := state.DnsConfig

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathCloudflareDNS)
	if err != nil {
		h.reportError(state, "fetch DNS HCL", err)
		return
	}

	existingKeys, err := hcleditor.ExistingDnsRecordKeys(src, zone)
	if err != nil {
		h.reportError(state, "read existing DNS keys", err)
		return
	}
	cfg.RecordKey = generateDnsRecordKey(cfg.Name, cfg.Type, existingKeys)

	modified, err := hcleditor.AddDnsRecord(src, zone, cfg)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.DnsBranchName("add", cfg.RecordKey)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(cloudflare): add DNS record %s", cfg.RecordKey)
	if err := h.gh.UpdateFile(ctx, branch, pathCloudflareDNS, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := h.resolveRequester(state)
	prTitle := fmt.Sprintf("Add DNS record: %s", cfg.RecordKey)
	prBody := ghclient.BuildDnsPRDescription("add", zone, cfg.RecordKey, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func (h *Handler) createDnsRemovePR(state *conversation.State) {
	ctx := context.Background()
	zone := state.TargetZone
	recordKey := state.TargetRecord

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathCloudflareDNS)
	if err != nil {
		h.reportError(state, "fetch DNS HCL", err)
		return
	}

	modified, err := hcleditor.RemoveDnsRecord(src, zone, recordKey)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.DnsBranchName("delete", recordKey)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(cloudflare): remove DNS record %s", recordKey)
	if err := h.gh.UpdateFile(ctx, branch, pathCloudflareDNS, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := h.resolveRequester(state)
	prTitle := fmt.Sprintf("Remove DNS record: %s", recordKey)
	prBody := ghclient.BuildDnsPRDescription("delete", zone, recordKey, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func (h *Handler) createDnsUpdatePR(state *conversation.State) {
	ctx := context.Background()
	zone := state.TargetZone
	recordKey := state.TargetRecord

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathCloudflareDNS)
	if err != nil {
		h.reportError(state, "fetch DNS HCL", err)
		return
	}

	modified, err := hcleditor.UpdateDnsRecord(src, zone, recordKey, state.DnsConfig)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.DnsBranchName("update", recordKey)
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := fmt.Sprintf("feat(cloudflare): update DNS record %s", recordKey)
	if err := h.gh.UpdateFile(ctx, branch, pathCloudflareDNS, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := h.resolveRequester(state)
	prTitle := fmt.Sprintf("Update DNS record: %s", recordKey)
	prBody := ghclient.BuildDnsPRDescription("settings", zone, recordKey, requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func (h *Handler) createOrgSettingsPR(state *conversation.State) {
	ctx := context.Background()

	src, fileSHA, err := h.gh.GetFileContent(ctx, pathGitHubOrg)
	if err != nil {
		h.reportError(state, "fetch org HCL", err)
		return
	}

	modified, err := hcleditor.UpdateOrgSettings(src, state.OrgConfig)
	if err != nil {
		h.reportError(state, "modify HCL", err)
		return
	}

	branch := ghclient.OrgSettingsBranchName()
	if err := h.gh.CreateBranchFromMain(ctx, branch); err != nil {
		h.reportError(state, "create branch", err)
		return
	}

	commitMsg := "feat(github): update organization settings"
	if err := h.gh.UpdateFile(ctx, branch, pathGitHubOrg, modified, fileSHA, commitMsg); err != nil {
		h.reportError(state, "commit file", err)
		return
	}

	requester := h.resolveRequester(state)
	prTitle := "Update GitHub organization settings"
	prBody := ghclient.BuildOrgSettingsPRDescription(requester, state.Justification)
	prURL, err := h.gh.CreatePR(ctx, branch, prTitle, prBody)
	if err != nil {
		h.reportError(state, "create PR", err)
		return
	}

	h.replyPR(state, prURL)
	h.store.Delete(state.ThreadTS)
}

func orgConfigEqual(a, b conversation.OrgConfig) bool {
	return a == b
}

func (h *Handler) fetchCloudflareHCL() ([]byte, error) {
	ctx := context.Background()
	src, _, err := h.gh.GetFileContent(ctx, pathCloudflareDNS)
	return src, err
}

func (h *Handler) fetchDnsRecordKeys(src []byte, zone string) []string {
	keys, err := hcleditor.ExistingDnsRecordKeys(src, zone)
	if err != nil {
		h.logger.Error("failed to extract dns record keys", "error", err)
		return nil
	}
	return keys
}

func (h *Handler) fetchDnsRecordOptions(src []byte, zone string) []DnsRecordOption {
	keys := h.fetchDnsRecordKeys(src, zone)
	opts := make([]DnsRecordOption, 0, len(keys))
	for _, k := range keys {
		cfg, err := hcleditor.ExtractDnsConfig(src, zone, k)
		if err != nil {
			h.logger.Error("failed to extract dns config for option", "error", err, "key", k)
			opts = append(opts, DnsRecordOption{Key: k, Label: k})
			continue
		}
		content := cfg.Content
		if len(content) > 30 {
			content = content[:30] + "..."
		}
		opts = append(opts, DnsRecordOption{
			Key:   k,
			Label: fmt.Sprintf("%s (%s) %s", cfg.Name, cfg.Type, content),
		})
	}
	return opts
}

// checkDnsAddConflict fetches live HCL and checks if adding a record with the
// given name and type would conflict with existing records in the zone.
// Returns an error message or empty string if no conflict.
func (h *Handler) checkDnsAddConflict(zone, name, typ string) string {
	src, err := h.fetchCloudflareHCL()
	if err != nil {
		h.logger.Error("failed to fetch HCL for conflict check", "error", err)
		return ""
	}
	keys, err := hcleditor.ExistingDnsRecordKeys(src, zone)
	if err != nil {
		h.logger.Error("failed to read existing dns keys for conflict check", "error", err)
		return ""
	}
	var existing []conversation.DnsConfig
	for _, k := range keys {
		cfg, err := hcleditor.ExtractDnsConfig(src, zone, k)
		if err != nil {
			continue
		}
		existing = append(existing, cfg)
	}
	return checkDnsConflict(name, typ, existing)
}

// checkDnsRecordStillExists fetches live HCL and verifies the record key still
// exists in the zone. Returns an error message or empty string if found.
func (h *Handler) checkDnsRecordStillExists(zone, key string) string {
	src, err := h.fetchCloudflareHCL()
	if err != nil {
		h.logger.Error("failed to fetch HCL for record existence check", "error", err)
		return ""
	}
	keys, err := hcleditor.ExistingDnsRecordKeys(src, zone)
	if err != nil {
		h.logger.Error("failed to read existing dns keys for existence check", "error", err)
		return ""
	}
	return checkDnsRecordExists(key, keys)
}

func (h *Handler) resolveRequester(state *conversation.State) string {
	requester := state.UserID
	if user, err := h.api.GetUserInfo(state.UserID); err != nil {
		h.logger.Error("failed to resolve slack user name", "error", err, "user_id", state.UserID)
	} else {
		requester = user.RealName
	}
	return requester
}

func dnsConfigEqual(a, b conversation.DnsConfig) bool {
	return a.Type == b.Type &&
		a.Name == b.Name &&
		a.Content == b.Content &&
		a.Proxied == b.Proxied &&
		a.Priority == b.Priority &&
		a.Comment == b.Comment
}

// repoConfigEqual compares two RepoConfig values for equality.
func repoConfigEqual(a, b conversation.RepoConfig) bool {
	if a.Description != b.Description || a.Visibility != b.Visibility || a.DefaultBranch != b.DefaultBranch {
		return false
	}
	if a.HasIssues != b.HasIssues || a.HasWiki != b.HasWiki {
		return false
	}
	if a.HasDiscussions != b.HasDiscussions || a.HasProjects != b.HasProjects {
		return false
	}
	if a.HomepageURL != b.HomepageURL {
		return false
	}
	if a.AllowAutoMerge != b.AllowAutoMerge || a.AllowUpdateBranch != b.AllowUpdateBranch || a.DeleteBranchOnMerge != b.DeleteBranchOnMerge {
		return false
	}
	if a.EnableBranchProtection != b.EnableBranchProtection {
		return false
	}
	if a.EnableBranchProtection && b.EnableBranchProtection {
		if a.RequiredReviews != b.RequiredReviews || a.DismissStaleReviews != b.DismissStaleReviews {
			return false
		}
		if a.RequireLinearHistory != b.RequireLinearHistory || a.RequireConversationResolution != b.RequireConversationResolution {
			return false
		}
	}
	if len(a.Topics) != len(b.Topics) {
		return false
	}
	for i := range a.Topics {
		if a.Topics[i] != b.Topics[i] {
			return false
		}
	}
	if len(a.TeamAccess) != len(b.TeamAccess) {
		return false
	}
	for k, v := range a.TeamAccess {
		if b.TeamAccess[k] != v {
			return false
		}
	}
	return true
}

func (h *Handler) reportError(state *conversation.State, step string, err error) {
	h.logger.Error("PR creation failed", "step", step, "error", err, "user", state.UserID)
	h.reply(state, slack.MsgOptionText(fmt.Sprintf("Something went wrong at %s: %v", step, err), false))
	h.store.Delete(state.ThreadTS)
}
