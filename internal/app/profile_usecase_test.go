package app

import (
	"fmt"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── func-field mocks for profile usecase tests ──

type stubIntegrationProfileRepo struct {
	CreateFn       func(profile *domain.IntegrationProfile) error
	FindByIDFn     func(id uint) (*domain.IntegrationProfile, error)
	FindByKeyFn    func(key string) (*domain.IntegrationProfile, error)
	ListFn         func() ([]domain.IntegrationProfile, error)
	UpdateFn       func(profile *domain.IntegrationProfile) error
	DeleteFn       func(id uint) error
}

func (s *stubIntegrationProfileRepo) Create(profile *domain.IntegrationProfile) error {
	if s.CreateFn != nil {
		return s.CreateFn(profile)
	}
	return nil
}

func (s *stubIntegrationProfileRepo) FindByID(id uint) (*domain.IntegrationProfile, error) {
	if s.FindByIDFn != nil {
		return s.FindByIDFn(id)
	}
	return &domain.IntegrationProfile{ID: id}, nil
}

func (s *stubIntegrationProfileRepo) FindByProfileKey(key string) (*domain.IntegrationProfile, error) {
	if s.FindByKeyFn != nil {
		return s.FindByKeyFn(key)
	}
	return nil, fmt.Errorf("not found")
}

func (s *stubIntegrationProfileRepo) List() ([]domain.IntegrationProfile, error) {
	if s.ListFn != nil {
		return s.ListFn()
	}
	return nil, nil
}

func (s *stubIntegrationProfileRepo) Update(profile *domain.IntegrationProfile) error {
	if s.UpdateFn != nil {
		return s.UpdateFn(profile)
	}
	return nil
}

func (s *stubIntegrationProfileRepo) Delete(id uint) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(id)
	}
	return nil
}

type stubDemandDocumentRepo struct {
	CountByProfileIDFn func(profileID uint) (int64, error)
}

func (s *stubDemandDocumentRepo) Create(_ *domain.DemandDocument) error { return nil }
func (s *stubDemandDocumentRepo) FindByID(_ uint) (*domain.DemandDocument, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubDemandDocumentRepo) List() ([]domain.DemandDocument, error)           { return nil, nil }
func (s *stubDemandDocumentRepo) ListUnassigned() ([]domain.DemandDocument, error) { return nil, nil }
func (s *stubDemandDocumentRepo) CountByProfileID(profileID uint) (int64, error) {
	if s.CountByProfileIDFn != nil {
		return s.CountByProfileIDFn(profileID)
	}
	return 0, nil
}
func (s *stubDemandDocumentRepo) CreateLine(_ *domain.DemandLine) error { return nil }
func (s *stubDemandDocumentRepo) FindLineByID(_ uint) (*domain.DemandLine, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubDemandDocumentRepo) ListLinesByDocument(_ uint) ([]domain.DemandLine, error) {
	return nil, nil
}
func (s *stubDemandDocumentRepo) UpdateLine(_ *domain.DemandLine) error { return nil }
func (s *stubDemandDocumentRepo) UpdateLineRoutingFields(_ uint, _, _, _ string) error {
	return nil
}
func (s *stubDemandDocumentRepo) UpdateBoundProfileSnapshot(_ uint, _ string) error { return nil }

type stubChannelSyncRepo struct {
	CountJobsByProfileIDFn func(profileID uint) (int64, error)
}

func (s *stubChannelSyncRepo) CreateJob(_ *domain.ChannelSyncJob) error { return nil }
func (s *stubChannelSyncRepo) FindJobByID(_ uint) (*domain.ChannelSyncJob, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubChannelSyncRepo) ListJobsByWave(_ uint) ([]domain.ChannelSyncJob, error) {
	return nil, nil
}
func (s *stubChannelSyncRepo) SaveJob(_ *domain.ChannelSyncJob) error    { return nil }
func (s *stubChannelSyncRepo) CreateItem(_ *domain.ChannelSyncItem) error { return nil }
func (s *stubChannelSyncRepo) SaveItem(_ *domain.ChannelSyncItem) error   { return nil }
func (s *stubChannelSyncRepo) ListItemsByJob(_ uint) ([]domain.ChannelSyncItem, error) {
	return nil, nil
}
func (s *stubChannelSyncRepo) AtomicCreateChannelSync(_ *domain.ChannelSyncJob, _ []*domain.ChannelSyncItem, _ *domain.BasisPinParam) error {
	return nil
}
func (s *stubChannelSyncRepo) CountJobsByProfileID(profileID uint) (int64, error) {
	if s.CountJobsByProfileIDFn != nil {
		return s.CountJobsByProfileIDFn(profileID)
	}
	return 0, nil
}

type stubProfileTemplateBindingRepo struct {
	CountByProfileIDFn func(profileID uint) (int64, error)
}

func (s *stubProfileTemplateBindingRepo) Create(_ *domain.IntegrationProfileTemplateBinding) error {
	return nil
}
func (s *stubProfileTemplateBindingRepo) ListByProfile(_ uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (s *stubProfileTemplateBindingRepo) FindDefaultByProfileAndType(_ uint, _ string) (*domain.IntegrationProfileTemplateBinding, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubProfileTemplateBindingRepo) Delete(_ uint) error { return nil }
func (s *stubProfileTemplateBindingRepo) CountByProfileID(profileID uint) (int64, error) {
	if s.CountByProfileIDFn != nil {
		return s.CountByProfileIDFn(profileID)
	}
	return 0, nil
}

type stubClosureDecisionRepo struct {
	CountByProfileIDFn func(profileID uint) (int64, error)
}

func (s *stubClosureDecisionRepo) Create(_ *domain.ChannelClosureDecisionRecord) error { return nil }
func (s *stubClosureDecisionRepo) AtomicCreate(_ []*domain.ChannelClosureDecisionRecord) error {
	return nil
}
func (s *stubClosureDecisionRepo) ListByFulfillmentLine(_ uint) ([]domain.ChannelClosureDecisionRecord, error) {
	return nil, nil
}
func (s *stubClosureDecisionRepo) ListByWave(_ uint) ([]domain.ChannelClosureDecisionRecord, error) {
	return nil, nil
}
func (s *stubClosureDecisionRepo) CountByProfileID(profileID uint) (int64, error) {
	if s.CountByProfileIDFn != nil {
		return s.CountByProfileIDFn(profileID)
	}
	return 0, nil
}

// ── validateExecutionReadiness tests ──

func TestValidateExecutionReadiness(t *testing.T) {
	t.Parallel()

	provider := NewRuntimeExecutorProviderWith(map[string]map[string]ChannelSyncExecutor{
		"api_push": {
			"my.connector": NewFakeExecutor(),
		},
		"document_export": {
			"eli.local_export": NewFakeExecutor(),
		},
	})

	tests := []struct {
		name    string
		input   dto.CreateProfileInput
		wantErr bool
	}{
		{
			name: "api_push with empty connectorKey is rejected",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "",
			},
			wantErr: true,
		},
		{
			name: "document_export with empty connectorKey is rejected",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "",
			},
			wantErr: true,
		},
		{
			name: "manual_confirmation with allowsManualClosure=false is rejected",
			input: dto.CreateProfileInput{
				ProfileKey:          "test",
				TrackingSyncMode:    "manual_confirmation",
				AllowsManualClosure: false,
			},
			wantErr: true,
		},
		{
			name: "api_push with valid connectorKey passes",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "my.connector",
			},
			wantErr: false,
		},
		{
			name: "api_push with unknown connectorKey is rejected",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "unknown.connector",
			},
			wantErr: true,
		},
		{
			name: "document_export with valid connectorKey passes",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "eli.local_export",
			},
			wantErr: false,
		},
		{
			name: "document_export with unknown connectorKey is rejected",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "unknown.connector",
			},
			wantErr: true,
		},
		{
			name: "manual_confirmation with allowsManualClosure=true passes",
			input: dto.CreateProfileInput{
				ProfileKey:          "test",
				TrackingSyncMode:    "manual_confirmation",
				AllowsManualClosure: true,
			},
			wantErr: false,
		},
		{
			name: "unsupported mode passes without any extra config",
			input: dto.CreateProfileInput{
				ProfileKey:       "test",
				TrackingSyncMode: "unsupported",
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateExecutionReadiness(tc.input, provider)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── CreateProfile validation integration tests ──

func TestCreateProfileValidatesExecutionReadiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   dto.CreateProfileInput
		wantErr bool
	}{
		{
			name: "api_push with empty connectorKey rejected at CreateProfile level",
			input: dto.CreateProfileInput{
				ProfileKey:       "test-profile",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "",
			},
			wantErr: true,
		},
		{
			name: "document_export with empty connectorKey rejected at CreateProfile level",
			input: dto.CreateProfileInput{
				ProfileKey:       "test-profile",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "",
			},
			wantErr: true,
		},
		{
			name: "manual_confirmation with allowsManualClosure=false rejected at CreateProfile level",
			input: dto.CreateProfileInput{
				ProfileKey:          "test-profile",
				TrackingSyncMode:    "manual_confirmation",
				AllowsManualClosure: false,
			},
			wantErr: true,
		},
		{
			name: "api_push with valid connectorKey passes CreateProfile validation",
			input: dto.CreateProfileInput{
				ProfileKey:       "test-profile",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "valid.connector",
			},
			wantErr: false,
		},
		{
			name: "document_export with unknown connectorKey rejected at CreateProfile level",
			input: dto.CreateProfileInput{
				ProfileKey:       "test-profile",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "unknown.connector",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			execProvider := NewRuntimeExecutorProviderWith(map[string]map[string]ChannelSyncExecutor{
				"api_push": {
					"valid.connector": NewFakeExecutor(),
				},
				"document_export": {
					"eli.local_export": NewFakeExecutor(),
				},
			})
			uc := NewProfileManagementUseCase(
				&stubIntegrationProfileRepo{},
				&stubDemandDocumentRepo{},
				&stubChannelSyncRepo{},
				&stubProfileTemplateBindingRepo{},
				&stubClosureDecisionRepo{},
				execProvider,
			)
			_, err := uc.CreateProfile(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── UpdateProfile validation tests ──

func TestUpdateProfileValidatesExecutionReadiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   dto.UpdateProfileInput
		wantErr bool
	}{
		{
			name: "api_push with empty connectorKey rejected at UpdateProfile level",
			input: dto.UpdateProfileInput{
				ID:               1,
				ProfileKey:       "test-profile",
				TrackingSyncMode: "api_push",
				ConnectorKey:     "",
			},
			wantErr: true,
		},
		{
			name: "manual_confirmation with allowsManualClosure=false rejected at UpdateProfile level",
			input: dto.UpdateProfileInput{
				ID:                  1,
				ProfileKey:          "test-profile",
				TrackingSyncMode:    "manual_confirmation",
				AllowsManualClosure: false,
			},
			wantErr: true,
		},
		{
			name: "document_export with unknown connectorKey rejected at UpdateProfile level",
			input: dto.UpdateProfileInput{
				ID:               1,
				ProfileKey:       "test-profile",
				TrackingSyncMode: "document_export",
				ConnectorKey:     "unknown.connector",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
				execProvider := NewRuntimeExecutorProviderWith(map[string]map[string]ChannelSyncExecutor{
					"api_push": {
						"valid.connector": NewFakeExecutor(),
					},
					"document_export": {
						"eli.local_export": NewFakeExecutor(),
					},
			})
			uc := NewProfileManagementUseCase(
				&stubIntegrationProfileRepo{},
				&stubDemandDocumentRepo{},
				&stubChannelSyncRepo{},
				&stubProfileTemplateBindingRepo{},
				&stubClosureDecisionRepo{},
				execProvider,
			)
			_, err := uc.UpdateProfile(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── DeleteProfile closureDecision gating tests ──

func TestDeleteProfileClosureDecisionGating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		closureCount int64
		wantErr      bool
	}{
		{
			name:         "closureDecision count > 0 rejects deletion",
			closureCount: 3,
			wantErr:      true,
		},
		{
			name:         "closureDecision count = 0 allows deletion (passes this check)",
			closureCount: 0,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
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
					CountByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
				},
				&stubChannelSyncRepo{
					CountJobsByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
				},
				&stubProfileTemplateBindingRepo{
					CountByProfileIDFn: func(_ uint) (int64, error) { return 0, nil },
				},
				&stubClosureDecisionRepo{
					CountByProfileIDFn: func(_ uint) (int64, error) { return tc.closureCount, nil },
				},
				nil,
			)

			err := uc.DeleteProfile(1)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tc.wantErr && !deleteCalled {
				t.Error("expected repo.Delete to be called when all checks pass")
			}
			if tc.wantErr && deleteCalled {
				t.Error("repo.Delete should not be called when closure decision check fails")
			}
		})
	}
}

func TestSeedDefaultProfilesUsesExecutableDefaults(t *testing.T) {
	t.Parallel()

	created := make(map[string]*domain.IntegrationProfile)
	provider := NewRuntimeExecutorProviderWith(map[string]map[string]ChannelSyncExecutor{
		"document_export": {
			"eli.local_export": NewFakeExecutor(),
		},
	})

	uc := NewProfileManagementUseCase(
		&stubIntegrationProfileRepo{
			FindByKeyFn: func(key string) (*domain.IntegrationProfile, error) {
				if p, ok := created[key]; ok {
					cp := *p
					return &cp, nil
				}
				return nil, fmt.Errorf("not found")
			},
			CreateFn: func(profile *domain.IntegrationProfile) error {
				cp := *profile
				created[profile.ProfileKey] = &cp
				return nil
			},
		},
		&stubDemandDocumentRepo{},
		&stubChannelSyncRepo{},
		&stubProfileTemplateBindingRepo{},
		&stubClosureDecisionRepo{},
		provider,
	)

	profiles, err := uc.SeedDefaultProfiles()
	if err != nil {
		t.Fatalf("SeedDefaultProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 seeded profiles, got %d", len(profiles))
	}

	retail, ok := created["retail_default"]
	if !ok {
		t.Fatal("retail_default was not created")
	}
	if retail.ConnectorKey != "eli.local_export" {
		t.Fatalf("retail_default connector_key = %q, want eli.local_export", retail.ConnectorKey)
	}
	if _, err := provider.Resolve(retail); err != nil {
		t.Fatalf("retail_default should resolve in runtime executor provider, got error: %v", err)
	}
}
