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
	ResponsePayload string
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
// It returns an error for any connector_key that has no registered executor.
type runtimeExecutorProvider struct{}

func NewRuntimeExecutorProvider() ExecutorProvider {
	return &runtimeExecutorProvider{}
}

func (p *runtimeExecutorProvider) Resolve(profile *domain.IntegrationProfile) (ChannelSyncExecutor, error) {
	if profile.ConnectorKey == "" {
		return nil, fmt.Errorf("no executor configured: integration profile %q has empty connector_key", profile.ProfileKey)
	}
	return nil, fmt.Errorf("no executor registered for connector_key %q (integration profile %q)", profile.ConnectorKey, profile.ProfileKey)
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
		ResponsePayload: `{"error":"all failed"}`,
		ErrorMessage:    "mock failure",
	}, nil
}
