package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock address repo ──

type mockAddressRepo struct {
	mu     sync.Mutex
	addrs  map[uint]*domain.CustomerAddress
	lastID uint
}

func newMockAddressRepo() *mockAddressRepo {
	return &mockAddressRepo{addrs: make(map[uint]*domain.CustomerAddress)}
}

func (m *mockAddressRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockAddressRepo) Create(ctx context.Context, addr *domain.CustomerAddress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.next()
	addr.ID = id
	cp := *addr
	m.addrs[id] = &cp
	return nil
}

func (m *mockAddressRepo) FindByID(ctx context.Context, id uint) (*domain.CustomerAddress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.addrs[id]
	if !ok {
		return nil, fmt.Errorf("address %d not found", id)
	}
	cp := *a
	return &cp, nil
}

func (m *mockAddressRepo) ListByProfile(ctx context.Context, profileID uint) ([]domain.CustomerAddress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.CustomerAddress
	for _, a := range m.addrs {
		if a.CustomerProfileID == profileID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAddressRepo) Update(ctx context.Context, addr *domain.CustomerAddress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.addrs[addr.ID]; ok {
		*existing = *addr
	}
	return nil
}

func (m *mockAddressRepo) SoftDelete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.addrs, id)
	return nil
}

func (m *mockAddressRepo) BulkUpdateProfileID(ctx context.Context, oldPID, newPID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.addrs {
		if a.CustomerProfileID == oldPID {
			a.CustomerProfileID = newPID
		}
	}
	return nil
}

func (m *mockAddressRepo) ClearDefaultByProfile(ctx context.Context, profileID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.addrs {
		if a.CustomerProfileID == profileID && a.IsDefault {
			a.IsDefault = false
		}
	}
	return nil
}

// ── tests ──

func TestCreateAddress(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	result, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
		RecipientName:     "Hina",
		Country:           "Kivotos",
		City:              "Gehenna",
		AddressLine1:      "Disciplinary Committee HQ",
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	if result.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if result.ValidationStatus != "unvalidated" {
		t.Errorf("expected default validation_status=unvalidated, got %q", result.ValidationStatus)
	}
}

func TestCreateAddressDefaultClearsPrevious(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	// create first default
	r1, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "First",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if !r1.IsDefault {
		t.Error("first address should be default")
	}

	// create second default — should clear the first
	r2, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Second",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	// verify first is no longer default
	a1, _ := addrRepo.FindByID(context.Background(), r1.ID)
	if a1.IsDefault {
		t.Error("first address should no longer be default")
	}
	if !r2.IsDefault {
		t.Error("second address should be default")
	}
}

func TestUpdateAddress(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Old",
		City:              "OldCity",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := uc.UpdateAddress(context.Background(), dto.UpdateAddressInput{
		ID:                created.ID,
		CustomerProfileID: 1,
		Label:             "New",
		City:              "NewCity",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Label != "New" || updated.City != "NewCity" {
		t.Errorf("update not applied: label=%q city=%q", updated.Label, updated.City)
	}
}

func TestDeleteAddress(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{CustomerProfileID: 1, Label: "Temp"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := uc.DeleteAddress(context.Background(), created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = uc.GetAddress(context.Background(), created.ID)
	if err == nil {
		t.Error("expected error getting deleted address")
	}
}

func TestBindAddressToLine(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	addr, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
		ValidationStatus:  "valid",
	})
	if err != nil {
		t.Fatalf("create address: %v", err)
	}

	// create a fulfillment line in the mock
	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1}
	if err := fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create line: %v", err)
	}

	bound, err := uc.BindAddressToLine(context.Background(), dto.BindAddressInput{
		FulfillmentLineID: fl.ID,
		CustomerAddressID: addr.ID,
	})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if bound.ID != addr.ID {
		t.Error("wrong address returned")
	}

	line, _ := fulfillRepo.FindByID(context.Background(), fl.ID)
	if line.CustomerAddressID == nil || *line.CustomerAddressID != addr.ID {
		t.Error("address not bound to line")
	}
	if line.AddressState != "ready" {
		t.Errorf("expected address_state=ready for valid address, got %q", line.AddressState)
	}
}

func TestBindInvalidAddressToLine(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	addr, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "BadAddr",
		ValidationStatus:  "invalid",
	})
	if err != nil {
		t.Fatalf("create address: %v", err)
	}

	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1}
	if err := fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create line: %v", err)
	}

	if _, err := uc.BindAddressToLine(context.Background(), dto.BindAddressInput{
		FulfillmentLineID: fl.ID,
		CustomerAddressID: addr.ID,
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}

	line, _ := fulfillRepo.FindByID(context.Background(), fl.ID)
	if line.AddressState != "invalid" {
		t.Errorf("expected address_state=invalid for invalid address, got %q", line.AddressState)
	}
}

func TestUnbindAddressFromLine(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	addr, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
		ValidationStatus:  "valid",
	})
	if err != nil {
		t.Fatalf("create address: %v", err)
	}

	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1}
	if err := fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create line: %v", err)
	}

	if _, err := uc.BindAddressToLine(context.Background(), dto.BindAddressInput{
		FulfillmentLineID: fl.ID,
		CustomerAddressID: addr.ID,
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if err := uc.UnbindAddressFromLine(context.Background(), fl.ID); err != nil {
		t.Fatalf("unbind: %v", err)
	}

	line, _ := fulfillRepo.FindByID(context.Background(), fl.ID)
	if line.CustomerAddressID != nil {
		t.Error("address should be nil after unbind")
	}
	if line.AddressState != "missing" {
		t.Errorf("expected address_state=missing after unbind, got %q", line.AddressState)
	}
}

func TestDeriveAddressState(t *testing.T) {
	t.Parallel()
	tests := []struct{ status, want string }{
		{"valid", "ready"},
		{"invalid", "invalid"},
		{"unvalidated", "ready"},
		{"unknown", "ready"},
	}
	for _, tt := range tests {
		got := deriveAddressState(tt.status)
		if got != tt.want {
			t.Errorf("deriveAddressState(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestListAddressesByProfile(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	for i := 0; i < 3; i++ {
		if _, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
			CustomerProfileID: 1,
			Label:             fmt.Sprintf("Addr %d", i),
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	// different profile
	uc.CreateAddress(context.Background(), dto.CreateAddressInput{CustomerProfileID: 2, Label: "Other"})

	list, err := uc.ListAddressesByProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 addresses for profile 1, got %d", len(list))
	}
}
