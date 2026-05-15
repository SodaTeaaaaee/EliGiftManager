package app

import (
	"fmt"
	"testing"
)

// TestDeleteProfileFKBlocking verifies that DeleteProfile returns a blocking error
// for each of the four FK reference checks: demand documents, channel sync jobs,
// template bindings, and closure decisions.
//
// Each sub-test sets exactly one FK count > 0 and all others to 0, confirming
// the specific check fires and repo.Delete is NOT called.
func TestDeleteProfileFKBlocking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		demandCount   int64
		syncCount     int64
		bindingCount  int64
		closureCount  int64
		wantErr       bool
		wantDeleteCalled bool
	}{
		{
			name:             "demand documents block deletion",
			demandCount:      2,
			syncCount:        0,
			bindingCount:     0,
			closureCount:     0,
			wantErr:          true,
			wantDeleteCalled: false,
		},
		{
			name:             "channel sync jobs block deletion",
			demandCount:      0,
			syncCount:        1,
			bindingCount:     0,
			closureCount:     0,
			wantErr:          true,
			wantDeleteCalled: false,
		},
		{
			name:             "template bindings block deletion",
			demandCount:      0,
			syncCount:        0,
			bindingCount:     3,
			closureCount:     0,
			wantErr:          true,
			wantDeleteCalled: false,
		},
		{
			name:             "closure decisions block deletion",
			demandCount:      0,
			syncCount:        0,
			bindingCount:     0,
			closureCount:     1,
			wantErr:          true,
			wantDeleteCalled: false,
		},
		{
			name:             "no references allows deletion",
			demandCount:      0,
			syncCount:        0,
			bindingCount:     0,
			closureCount:     0,
			wantErr:          false,
			wantDeleteCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deleteCalled := false
			uc := NewProfileManagementUseCase(
				&stubIntegrationProfileRepo{
					DeleteFn: func(id uint) error {
						deleteCalled = true
						return nil
					},
				},
				&stubDemandDocumentRepo{
					CountByProfileIDFn: func(_ uint) (int64, error) { return tc.demandCount, nil },
				},
				&stubChannelSyncRepo{
					CountJobsByProfileIDFn: func(_ uint) (int64, error) { return tc.syncCount, nil },
				},
				&stubProfileTemplateBindingRepo{
					CountByProfileIDFn: func(_ uint) (int64, error) { return tc.bindingCount, nil },
				},
				&stubClosureDecisionRepo{
					CountByProfileIDFn: func(_ uint) (int64, error) { return tc.closureCount, nil },
				},
				nil,
			)

			err := uc.DeleteProfile(1)

			if tc.wantErr && err == nil {
				t.Errorf("expected blocking error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tc.wantDeleteCalled && !deleteCalled {
				t.Error("expected repo.Delete to be called when no FK references exist")
			}
			if !tc.wantDeleteCalled && deleteCalled {
				t.Error("repo.Delete must not be called when FK references exist")
			}
		})
	}
}

// TestDeleteProfileDemandErrorPropagates verifies that a repo error during the
// demand-count check is propagated rather than silently ignored.
func TestDeleteProfileDemandErrorPropagates(t *testing.T) {
	t.Parallel()

	uc := NewProfileManagementUseCase(
		&stubIntegrationProfileRepo{},
		&stubDemandDocumentRepo{
			CountByProfileIDFn: func(_ uint) (int64, error) {
				return 0, errMockRepoFailure
			},
		},
		&stubChannelSyncRepo{},
		&stubProfileTemplateBindingRepo{},
		&stubClosureDecisionRepo{},
		nil,
	)

	err := uc.DeleteProfile(1)
	if err == nil {
		t.Fatal("expected error when demand count repo fails, got nil")
	}
}

// TestDeleteProfileSyncErrorPropagates verifies that a repo error during the
// channel-sync-count check is propagated.
func TestDeleteProfileSyncErrorPropagates(t *testing.T) {
	t.Parallel()

	uc := NewProfileManagementUseCase(
		&stubIntegrationProfileRepo{},
		&stubDemandDocumentRepo{
			CountByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
		},
		&stubChannelSyncRepo{
			CountJobsByProfileIDFn: func(_ uint) (int64, error) {
				return 0, errMockRepoFailure
			},
		},
		&stubProfileTemplateBindingRepo{},
		&stubClosureDecisionRepo{},
		nil,
	)

	err := uc.DeleteProfile(1)
	if err == nil {
		t.Fatal("expected error when channel sync count repo fails, got nil")
	}
}

// TestDeleteProfileBindingErrorPropagates verifies that a repo error during the
// template-binding-count check is propagated.
func TestDeleteProfileBindingErrorPropagates(t *testing.T) {
	t.Parallel()

	uc := NewProfileManagementUseCase(
		&stubIntegrationProfileRepo{},
		&stubDemandDocumentRepo{
			CountByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
		},
		&stubChannelSyncRepo{
			CountJobsByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
		},
		&stubProfileTemplateBindingRepo{
			CountByProfileIDFn: func(_ uint) (int64, error) {
				return 0, errMockRepoFailure
			},
		},
		&stubClosureDecisionRepo{},
		nil,
	)

	err := uc.DeleteProfile(1)
	if err == nil {
		t.Fatal("expected error when template binding count repo fails, got nil")
	}
}

// errMockRepoFailure is a sentinel error used by FK error-propagation tests.
var errMockRepoFailure = fmt.Errorf("mock: repo failure")
