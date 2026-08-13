package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
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

func (m *mockAddressRepo) FeatureEnabled(context.Context, string) (bool, error) { return true, nil }
func (m *mockAddressRepo) RequireFeature(context.Context, string) error         { return nil }

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

	profileID := uint(1)
	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1, CustomerProfileID: &profileID}
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

	profileID := uint(1)
	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1, CustomerProfileID: &profileID}
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

	profileID := uint(1)
	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1, CustomerProfileID: &profileID}
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
		{"unvalidated", "missing"},
		{"unknown", "missing"},
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

func TestCreateAddressRejectsZeroProfile(t *testing.T) {
	t.Parallel()
	uc := NewAddressManagementUseCase(newMockAddressRepo(), newMockFulfillRepo())

	_, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 0,
		Label:             "Home",
	})
	if err == nil {
		t.Fatal("expected error creating address on empty profile")
	}
}

func TestBindAddressToLineProfileMismatch(t *testing.T) {
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

	otherProfileID := uint(2)
	fl := &domain.FulfillmentLine{WaveID: 1, Quantity: 1, CustomerProfileID: &otherProfileID}
	if err := fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create line: %v", err)
	}

	if _, err := uc.BindAddressToLine(context.Background(), dto.BindAddressInput{
		FulfillmentLineID: fl.ID,
		CustomerAddressID: addr.ID,
	}); err == nil {
		t.Fatal("expected error binding address across different profiles")
	}

	line, _ := fulfillRepo.FindByID(context.Background(), fl.ID)
	if line.CustomerAddressID != nil {
		t.Error("address should not be bound when profiles mismatch")
	}
}

func TestUpdateAddressKeepsProfileWhenInputZero(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	uc := NewAddressManagementUseCase(addrRepo, newMockFulfillRepo())

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
		City:              "Gehenna",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := uc.UpdateAddress(context.Background(), dto.UpdateAddressInput{
		ID:                created.ID,
		CustomerProfileID: 0,
		Label:             "Home 2",
		City:              "Gehenna",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.CustomerProfileID != 1 {
		t.Errorf("expected profile 1 to be kept, got %d", updated.CustomerProfileID)
	}
	if updated.Label != "Home 2" {
		t.Errorf("expected label update, got %q", updated.Label)
	}
}

func TestUpdateAddressRejectsProfileReassign(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	uc := NewAddressManagementUseCase(addrRepo, newMockFulfillRepo())

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := uc.UpdateAddress(context.Background(), dto.UpdateAddressInput{
		ID:                created.ID,
		CustomerProfileID: 2,
		Label:             "Moved",
	}); err == nil {
		t.Fatal("expected error reassigning address to a different profile")
	}

	got, err := addrRepo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.CustomerProfileID != 1 {
		t.Errorf("profile should remain 1, got %d", got.CustomerProfileID)
	}
}

func TestUpdateAddressDefaultClearsActualProfile(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	uc := NewAddressManagementUseCase(addrRepo, newMockFulfillRepo())

	zeroDefault := &domain.CustomerAddress{
		CustomerProfileID: 0,
		Label:             "ZeroDefault",
		IsDefault:         true,
	}
	if err := addrRepo.Create(context.Background(), zeroDefault); err != nil {
		t.Fatalf("create zero-profile default: %v", err)
	}

	first, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "First",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}

	second, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Second",
		IsDefault:         false,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	updated, err := uc.UpdateAddress(context.Background(), dto.UpdateAddressInput{
		ID:                second.ID,
		CustomerProfileID: 0,
		Label:             "Second",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.CustomerProfileID != 1 {
		t.Errorf("expected profile 1 to be kept, got %d", updated.CustomerProfileID)
	}
	if !updated.IsDefault {
		t.Error("second address should be default")
	}

	a1, _ := addrRepo.FindByID(context.Background(), first.ID)
	if a1.IsDefault {
		t.Error("first address on profile 1 should no longer be default")
	}

	zero, _ := addrRepo.FindByID(context.Background(), zeroDefault.ID)
	if !zero.IsDefault {
		t.Error("default on profile 0 should not be cleared")
	}
}

func pinSQLiteConn(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
}

func failCustomerAddressCreates(t *testing.T, db *gorm.DB) {
	t.Helper()
	const name = "test_fail_customer_address_create"
	if err := db.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		table := tx.Statement.Table
		if tx.Statement.Schema != nil && table == "" {
			table = tx.Statement.Schema.Table
		}
		if table == "customer_addresses" {
			_ = tx.AddError(fmt.Errorf("forced address persist failure"))
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(name) })
}

func failCustomerAddressSaves(t *testing.T, db *gorm.DB) {
	t.Helper()
	const name = "test_fail_customer_address_save"
	if err := db.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		table := tx.Statement.Table
		if tx.Statement.Schema != nil && table == "" {
			table = tx.Statement.Schema.Table
		}
		if table != "customer_addresses" {
			return
		}
		dest, ok := tx.Statement.Dest.(*persistence.CustomerAddress)
		if ok && dest.ID != 0 {
			_ = tx.AddError(fmt.Errorf("forced address save failure"))
		}
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Update().Remove(name) })
}

func TestCreateAddressDefaultWriteRollsBackTogether(t *testing.T) {
	db := setupTestDB(t)
	pinSQLiteConn(t, db)
	ctx := context.Background()
	profileID := createTestProfile(t, db, "DefaultRollback")
	addrRepo := infra.NewAddressRepository(db)
	uc := NewAddressManagementUseCase(addrRepo, infra.NewFulfillmentRepository(db), db)

	first, err := uc.CreateAddress(ctx, dto.CreateAddressInput{
		CustomerProfileID: profileID,
		Label:             "First",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create first default: %v", err)
	}

	failCustomerAddressCreates(t, db)
	if _, err := uc.CreateAddress(ctx, dto.CreateAddressInput{
		CustomerProfileID: profileID,
		Label:             "Second",
		IsDefault:         true,
	}); err == nil {
		t.Fatal("expected second default create to fail")
	}

	got, err := addrRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("reload first: %v", err)
	}
	if !got.IsDefault {
		t.Fatal("first address must remain default when ClearDefault+Create rolls back together")
	}
}

func TestCreateAddressDefaultWriteUsesNestedSavepoint(t *testing.T) {
	db := setupTestDB(t)
	pinSQLiteConn(t, db)
	ctx := context.Background()
	profileID := createTestProfile(t, db, "NestedSavepoint")
	addrRepo := infra.NewAddressRepository(db)
	uc := NewAddressManagementUseCase(addrRepo, infra.NewFulfillmentRepository(db), db)

	first, err := uc.CreateAddress(ctx, dto.CreateAddressInput{
		CustomerProfileID: profileID,
		Label:             "First",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create first default: %v", err)
	}

	failCustomerAddressCreates(t, db)
	outerErr := db.Transaction(func(tx *gorm.DB) error {
		inner := NewAddressManagementUseCase(infra.NewAddressRepository(tx), infra.NewFulfillmentRepository(tx), tx)
		if _, err := inner.CreateAddress(ctx, dto.CreateAddressInput{
			CustomerProfileID: profileID,
			Label:             "Second",
			IsDefault:         true,
		}); err == nil {
			t.Fatal("expected inner create to fail")
		}
		return nil
	})
	if outerErr != nil {
		t.Fatalf("outer transaction: %v", outerErr)
	}

	got, err := addrRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("reload first: %v", err)
	}
	if !got.IsDefault {
		t.Fatal("nested savepoint must roll back ClearDefault without committing through the outer tx")
	}
}

func TestUpdateAddressDefaultWriteRollsBackTogether(t *testing.T) {
	db := setupTestDB(t)
	pinSQLiteConn(t, db)
	ctx := context.Background()
	profileID := createTestProfile(t, db, "UpdateDefaultRollback")
	addrRepo := infra.NewAddressRepository(db)
	uc := NewAddressManagementUseCase(addrRepo, infra.NewFulfillmentRepository(db), db)

	first, err := uc.CreateAddress(ctx, dto.CreateAddressInput{
		CustomerProfileID: profileID,
		Label:             "First",
		IsDefault:         true,
	})
	if err != nil {
		t.Fatalf("create first default: %v", err)
	}
	second, err := uc.CreateAddress(ctx, dto.CreateAddressInput{
		CustomerProfileID: profileID,
		Label:             "Second",
		IsDefault:         false,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	failCustomerAddressSaves(t, db)
	if _, err := uc.UpdateAddress(ctx, dto.UpdateAddressInput{
		ID:                second.ID,
		CustomerProfileID: profileID,
		Label:             "Second",
		IsDefault:         true,
	}); err == nil {
		t.Fatal("expected default promotion update to fail")
	}

	gotFirst, err := addrRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("reload first: %v", err)
	}
	if !gotFirst.IsDefault {
		t.Fatal("first address must remain default when ClearDefault+Update rolls back together")
	}
	gotSecond, err := addrRepo.FindByID(ctx, second.ID)
	if err != nil {
		t.Fatalf("reload second: %v", err)
	}
	if gotSecond.IsDefault {
		t.Fatal("second address must not become default after rolled-back promotion")
	}
}
