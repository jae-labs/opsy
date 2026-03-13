package conversation

import "sync"

// Phase represents the current step in the conversation flow.
type Phase int

const (
	PhaseIdle Phase = iota
	PhaseCategorySelected
	PhaseResourceSelected
	PhaseActionSelected
	PhaseWizardStep1
	PhaseWizardStep2
	PhaseWizardStep3
	PhaseConfirming
	PhaseCreatingPR
)

// State tracks a single flow's position in the conversation wizard.
// Each thread is one flow.
type State struct {
	Phase        Phase
	Category     string // "github", "cloudflare", "doppler"
	ResourceType string // "repo", "user_management", "settings"
	ChannelID    string // DM channel
	ThreadTS     string // thread parent timestamp (= user's original message ts)
	UserID        string
	ActionType    string // "add", "delete", "settings"
	TargetRepo    string // repo name for delete/settings flows
	TargetZone    string // dns zone for cloudflare flows
	TargetRecord  string // dns record key for update/delete flows
	Justification  string
	AvailableTeams []string // teams fetched from terraform, used to parse team role blocks
	RepoConfig     RepoConfig
	DnsConfig     DnsConfig
	OrgConfig     OrgConfig
}

// Store is a concurrency-safe in-memory store keyed by thread timestamp.
type Store struct {
	mu     sync.Mutex
	states map[string]*State // key = threadTS
}

func NewStore() *Store {
	return &Store{states: make(map[string]*State)}
}

// Get returns the state for a thread, or nil if not found.
func (s *Store) Get(threadTS string) *State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[threadTS]
}

// Create starts a new flow for a thread.
func (s *Store) Create(threadTS, channelID, userID string) *State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := &State{
		Phase:     PhaseIdle,
		ChannelID: channelID,
		ThreadTS:  threadTS,
		UserID:    userID,
	}
	s.states[threadTS] = st
	return st
}

// Delete removes a flow.
func (s *Store) Delete(threadTS string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, threadTS)
}
