package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ChannelSyncItemResult captures the outcome for a single item.
type ChannelSyncItemResult struct {
	ItemID       uint
	Status       string // "success" or "failed"
	ErrorMessage string
}

// ChannelSyncExecutionResult captures the aggregate outcome of a sync execution.
type ChannelSyncExecutionResult struct {
	Items           []ChannelSyncItemResult
	AggregateStatus string // "success", "partial_success", "failed"
	RequestPayload  string // real execution content snapshot (export payload, API request body)
	ResponsePayload string // execution result summary (output path, external response)
	ErrorMessage    string
}

// ChannelSyncExecutor drives the actual sync action for a job.
type ChannelSyncExecutor interface {
	Execute(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem, profile *domain.IntegrationProfile) (*ChannelSyncExecutionResult, error)
}

// ExecutorProvider resolves a ChannelSyncExecutor from an IntegrationProfile at runtime.
// The provider encapsulates the connector_key → executor mapping so that the
// production wiring never defaults to a fake success executor.
type ExecutorProvider interface {
	Resolve(profile *domain.IntegrationProfile) (ChannelSyncExecutor, error)
}

// runtimeExecutorProvider is the real production wiring.
// It resolves executors from a two-level registry keyed by
// tracking_sync_mode → connector_key so that mode mismatches
// are caught before execution.
type runtimeExecutorProvider struct {
	registry map[string]map[string]ChannelSyncExecutor
}

// NewRuntimeExecutorProvider returns a provider with an empty registry.
func NewRuntimeExecutorProvider() ExecutorProvider {
	return &runtimeExecutorProvider{registry: make(map[string]map[string]ChannelSyncExecutor)}
}

// NewRuntimeExecutorProviderWith returns a provider pre-populated with the
// given tracking_sync_mode → connector_key → executor mapping.
func NewRuntimeExecutorProviderWith(registry map[string]map[string]ChannelSyncExecutor) ExecutorProvider {
	return &runtimeExecutorProvider{registry: registry}
}

func (p *runtimeExecutorProvider) Resolve(profile *domain.IntegrationProfile) (ChannelSyncExecutor, error) {
	if profile.ConnectorKey == "" {
		return nil, fmt.Errorf("no executor configured: integration profile %q has empty connector_key", profile.ProfileKey)
	}
	modeExecs, ok := p.registry[profile.TrackingSyncMode]
	if !ok {
		return nil, fmt.Errorf("no executor registered for tracking_sync_mode %q (integration profile %q, connector_key %q)", profile.TrackingSyncMode, profile.ProfileKey, profile.ConnectorKey)
	}
	exec, ok := modeExecs[profile.ConnectorKey]
	if !ok {
		return nil, fmt.Errorf("no executor registered for connector_key %q under tracking_sync_mode %q (integration profile %q)", profile.ConnectorKey, profile.TrackingSyncMode, profile.ProfileKey)
	}
	return exec, nil
}



// fakeExecutor is a test implementation that marks all items as "success".
type fakeExecutor struct{}

func NewFakeExecutor() ChannelSyncExecutor { return &fakeExecutor{} }

// StaticExecutorProvider is a test helper that always resolves to the given executor.
type StaticExecutorProvider struct {
	Executor ChannelSyncExecutor
}

func (p *StaticExecutorProvider) Resolve(profile *domain.IntegrationProfile) (ChannelSyncExecutor, error) {
	return p.Executor, nil
}

func (e *fakeExecutor) Execute(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem, profile *domain.IntegrationProfile) (*ChannelSyncExecutionResult, error) {
	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{
			ItemID:       it.ID,
			Status:       "success",
			ErrorMessage: "",
		}
	}
	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "success",
		RequestPayload:  fmt.Sprintf(`{"kind":"fake_success","items":%d}`, len(items)),
		ResponsePayload: `{"status":"ok"}`,
	}, nil
}

// fakePartialExecutor simulates partial success (first item fails, rest succeed).
type fakePartialExecutor struct{}

func NewFakePartialExecutor() ChannelSyncExecutor { return &fakePartialExecutor{} }

func (e *fakePartialExecutor) Execute(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem, profile *domain.IntegrationProfile) (*ChannelSyncExecutionResult, error) {
	results := make([]ChannelSyncItemResult, len(items))
	hasSuccess := false
	hasFailure := false
	for i, it := range items {
		if i == 0 {
			results[i] = ChannelSyncItemResult{ItemID: it.ID, Status: "failed", ErrorMessage: "mock failure"}
			hasFailure = true
		} else {
			results[i] = ChannelSyncItemResult{ItemID: it.ID, Status: "success"}
			hasSuccess = true
		}
	}
	agg := "partial_success"
	if hasFailure && !hasSuccess {
		agg = "failed"
	} else if hasSuccess && !hasFailure {
		agg = "success"
	}
	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: agg,
		RequestPayload:  fmt.Sprintf(`{"kind":"fake_partial","items":%d}`, len(items)),
		ResponsePayload: `{"status":"partial"}`,
		ErrorMessage:    "mock failure",
	}, nil
}

// fakeFailingExecutor simulates total failure.
type fakeFailingExecutor struct{}

func NewFakeFailingExecutor() ChannelSyncExecutor { return &fakeFailingExecutor{} }

func (e *fakeFailingExecutor) Execute(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem, profile *domain.IntegrationProfile) (*ChannelSyncExecutionResult, error) {
	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{ItemID: it.ID, Status: "failed", ErrorMessage: "mock failure"}
	}
	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "failed",
		RequestPayload:  fmt.Sprintf(`{"kind":"fake_failing","items":%d}`, len(items)),
		ResponsePayload: `{"error":"all failed"}`,
		ErrorMessage:    "mock failure",
	}, nil
}
