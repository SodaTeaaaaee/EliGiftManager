package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type mockProfileRepoForIdentity struct {
	mu         sync.Mutex
	profiles   map[uint]*domain.CustomerProfile
	identities map[uint]*domain.CustomerIdentity
	lastID     uint
}

func newMockProfileRepoForIdentity() *mockProfileRepoForIdentity {
	return &mockProfileRepoForIdentity{
		profiles:   make(map[uint]*domain.CustomerProfile),
		identities: make(map[uint]*domain.CustomerIdentity),
	}
}

func (m *mockProfileRepoForIdentity) next() uint { m.lastID++; return m.lastID }

func (m *mockProfileRepoForIdentity) Create(ctx context.Context, p *domain.CustomerProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.ID = m.next()
	cp := *p
	m.profiles[p.ID] = &cp
	return nil
}

func (m *mockProfileRepoForIdentity) FindByID(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}

func (m *mockProfileRepoForIdentity) List(ctx context.Context) ([]domain.CustomerProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.CustomerProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		out = append(out, *p)
	}
	return out, nil
}

func (m *mockProfileRepoForIdentity) CreateIdentity(ctx context.Context, id *domain.CustomerIdentity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id.ID = m.next()
	cp := *id
	m.identities[id.ID] = &cp
	return nil
}

func (m *mockProfileRepoForIdentity) ListIdentitiesByProfile(ctx context.Context, pid uint) ([]domain.CustomerIdentity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.CustomerIdentity
	for _, i := range m.identities {
		if i.CustomerProfileID == pid {
			out = append(out, *i)
		}
	}
	return out, nil
}

func (m *mockProfileRepoForIdentity) FindIdentityByPlatformAndValue(ctx context.Context, platform, value string) (*domain.CustomerIdentity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range m.identities {
		if i.IdentityPlatform == platform && i.IdentityValue == value {
			cp := *i
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockProfileRepoForIdentity) UpdateIdentityProfileID(ctx context.Context, id, newPID uint) error {
	return nil
}

func (m *mockProfileRepoForIdentity) BulkUpdateIdentityProfileID(ctx context.Context, ids []uint, newPID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if i, ok := m.identities[id]; ok {
			i.CustomerProfileID = newPID
		}
	}
	return nil
}

func (m *mockProfileRepoForIdentity) SoftDelete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, id)
	return nil
}

func (m *mockProfileRepoForIdentity) DeleteIdentity(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.identities, id)
	return nil
}

func (m *mockProfileRepoForIdentity) Update(ctx context.Context, p *domain.CustomerProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[p.ID] = p
	return nil
}

func TestResolveOrCreateNewProfile(t *testing.T) {
	t.Parallel()
	repo := newMockProfileRepoForIdentity()
	svc := NewIdentityResolutionService(repo)

	pid, err := svc.ResolveOrCreateProfile(context.Background(), "test_platform", "user_123", "platform_uid")
	if err != nil {
		t.Fatalf("ResolveOrCreateProfile: %v", err)
	}
	if pid == 0 {
		t.Error("expected non-zero profile ID")
	}

	// Verify profile and identity were created
	profile, err := repo.FindByID(context.Background(), pid)
	if err != nil {
		t.Fatalf("find profile: %v", err)
	}
	if profile.DisplayName != "user_123" {
		t.Errorf("expected DisplayName=user_123, got %q", profile.DisplayName)
	}

	identities, err := repo.ListIdentitiesByProfile(context.Background(), pid)
	if err != nil {
		t.Fatalf("list identities: %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
	if identities[0].IdentityPlatform != "test_platform" {
		t.Errorf("expected platform=test_platform, got %q", identities[0].IdentityPlatform)
	}
}

func TestResolveOrCreateExistingProfile(t *testing.T) {
	t.Parallel()
	repo := newMockProfileRepoForIdentity()
	svc := NewIdentityResolutionService(repo)

	// Create first
	pid1, err := svc.ResolveOrCreateProfile(context.Background(), "platform_a", "user_42", "platform_uid")
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}

	// Resolve again — should return same profile
	pid2, err := svc.ResolveOrCreateProfile(context.Background(), "platform_a", "user_42", "platform_uid")
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}

	if pid1 != pid2 {
		t.Errorf("expected same profile ID: %d vs %d", pid1, pid2)
	}

	// Should have exactly 1 profile
	profiles, _ := repo.List(context.Background())
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}
}

func TestResolveIdentityStrategy(t *testing.T) {
	t.Parallel()
	tests := []struct{ input, want string }{
		{"email", "email"},
		{"external_buyer_id", "external_buyer_id"},
		{"platform_uid", "platform_uid"},
		{"unknown", "platform_uid"},
	}
	for _, tt := range tests {
		got := ResolveIdentityStrategy(tt.input)
		if got != tt.want {
			t.Errorf("ResolveIdentityStrategy(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveOrCreateEmptyIdentity(t *testing.T) {
	t.Parallel()
	repo := newMockProfileRepoForIdentity()
	svc := NewIdentityResolutionService(repo)

	_, err := svc.ResolveOrCreateProfile(context.Background(), "", "value", "platform_uid")
	if err == nil {
		t.Error("expected error for empty platform")
	}

	_, err = svc.ResolveOrCreateProfile(context.Background(), "platform", "", "platform_uid")
	if err == nil {
		t.Error("expected error for empty value")
	}
}
