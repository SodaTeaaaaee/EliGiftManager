package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── Mocks specific to AllocationPolicyUseCase tests ──

// policyWaveRepo is a mock WaveRepository that supports returning participants.
type policyWaveRepo struct {
	mu           sync.Mutex
	waves        map[uint]*domain.Wave
	participants map[uint][]domain.WaveParticipantSnapshot // waveID -> snapshots
	lastID       uint
}

func newPolicyWaveRepo() *policyWaveRepo {
	return &policyWaveRepo{
		waves:        make(map[uint]*domain.Wave),
		participants: make(map[uint][]domain.WaveParticipantSnapshot),
	}
}

func (m *policyWaveRepo) next() uint { m.lastID++; return m.lastID }

func (m *policyWaveRepo) Create(ctx context.Context, wave *domain.Wave) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	wave.ID = m.next()
	cp := *wave
	m.waves[wave.ID] = &cp
	return nil
}

func (m *policyWaveRepo) FindByID(ctx context.Context, id uint) (*domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.waves[id]
	if !ok {
		return nil, fmt.Errorf("wave %d not found", id)
	}
	cp := *w
	return &cp, nil
}

func (m *policyWaveRepo) FindByWaveNo(ctx context.Context, waveNo string) (*domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.waves {
		if w.WaveNo == waveNo {
			cp := *w
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("wave %q not found", waveNo)
}

func (m *policyWaveRepo) List(ctx context.Context) ([]domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Wave, 0, len(m.waves))
	for _, w := range m.waves {
		out = append(out, *w)
	}
	return out, nil
}

func (m *policyWaveRepo) ListPaginated(ctx context.Context, _, _ int) ([]domain.Wave, int64, error) {
	all, err := m.List(context.Background())
	return all, int64(len(all)), err
}

func (m *policyWaveRepo) AddParticipant(ctx context.Context, snap *domain.WaveParticipantSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap.ID = m.next()
	m.participants[snap.WaveID] = append(m.participants[snap.WaveID], *snap)
	return nil
}

func (m *policyWaveRepo) ListParticipantsByWave(ctx context.Context, waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pp := m.participants[waveID]
	out := make([]domain.WaveParticipantSnapshot, len(pp))
	copy(out, pp)
	return out, nil
}

func (m *policyWaveRepo) ListParticipantsByProfile(ctx context.Context, profileID uint) ([]domain.WaveParticipantSnapshot, error) {
	panic("not implemented")
}

func (m *policyWaveRepo) UpdateLifecycle(ctx context.Context, _ uint, _ string, _ string) error {
	return nil
}

func (m *policyWaveRepo) UpdateParticipantProfileID(ctx context.Context, oldPID, newPID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for waveID, snaps := range m.participants {
		for i := range snaps {
			if snaps[i].CustomerProfileID == oldPID {
				snaps[i].CustomerProfileID = newPID
				n++
			}
		}
		m.participants[waveID] = snaps
	}
	return n, nil
}

func (m *policyWaveRepo) DeleteParticipantsByWave(ctx context.Context, waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.participants, waveID)
	return nil
}

func (m *policyWaveRepo) CountByDatePrefix(ctx context.Context, prefix string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, w := range m.waves {
		if len(w.WaveNo) >= len(prefix) && w.WaveNo[:len(prefix)] == prefix {
			count++
		}
	}
	return count, nil
}

// policyRuleRepo is a mock AllocationPolicyRuleRepository with full CRUD.
type policyRuleRepo struct {
	mu     sync.Mutex
	rules  map[uint]*domain.AllocationPolicyRule
	lastID uint
}

func newPolicyRuleRepo() *policyRuleRepo {
	return &policyRuleRepo{rules: make(map[uint]*domain.AllocationPolicyRule)}
}

func (m *policyRuleRepo) next() uint { m.lastID++; return m.lastID }

func (m *policyRuleRepo) Create(ctx context.Context, rule *domain.AllocationPolicyRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rule.ID = m.next()
	cp := *rule
	m.rules[rule.ID] = &cp
	return nil
}

func (m *policyRuleRepo) FindByID(ctx context.Context, id uint) (*domain.AllocationPolicyRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rules[id]
	if !ok {
		return nil, fmt.Errorf("rule %d not found", id)
	}
	cp := *r
	return &cp, nil
}

func (m *policyRuleRepo) ListByWave(ctx context.Context, waveID uint) ([]domain.AllocationPolicyRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.AllocationPolicyRule
	for _, r := range m.rules {
		if r.WaveID == waveID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (m *policyRuleRepo) Update(ctx context.Context, rule *domain.AllocationPolicyRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[rule.ID]; !ok {
		return fmt.Errorf("rule %d not found", rule.ID)
	}
	cp := *rule
	m.rules[rule.ID] = &cp
	return nil
}

func (m *policyRuleRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[id]; !ok {
		return fmt.Errorf("rule %d not found", id)
	}
	delete(m.rules, id)
	return nil
}

func (m *policyRuleRepo) DeleteByWave(ctx context.Context, waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, r := range m.rules {
		if r.WaveID == waveID {
			delete(m.rules, id)
		}
	}
	return nil
}

// policyAdjRepo is a minimal mock FulfillmentAdjustmentRepository.
type policyAdjRepo struct {
	mu     sync.Mutex
	adjs   []domain.FulfillmentAdjustment
	lastID uint
}

func newPolicyAdjRepo() *policyAdjRepo {
	return &policyAdjRepo{}
}

func (m *policyAdjRepo) next() uint { m.lastID++; return m.lastID }

func (m *policyAdjRepo) Create(ctx context.Context, adj *domain.FulfillmentAdjustment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	adj.ID = m.next()
	m.adjs = append(m.adjs, *adj)
	return nil
}

func (m *policyAdjRepo) ListByWave(ctx context.Context, waveID uint) ([]domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FulfillmentAdjustment
	for _, a := range m.adjs {
		if a.WaveID == waveID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *policyAdjRepo) ListByFulfillmentLine(ctx context.Context, fulfillmentLineID uint) ([]domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FulfillmentAdjustment
	for _, a := range m.adjs {
		if a.FulfillmentLineID != nil && *a.FulfillmentLineID == fulfillmentLineID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *policyAdjRepo) FindByID(ctx context.Context, id uint) (*domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.adjs {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *policyAdjRepo) Update(ctx context.Context, adj *domain.FulfillmentAdjustment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.adjs {
		if m.adjs[i].ID == adj.ID {
			m.adjs[i] = *adj
			return nil
		}
	}
	return fmt.Errorf("adjustment %d not found", adj.ID)
}

func (m *policyAdjRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, a := range m.adjs {
		if a.ID == id {
			m.adjs = append(m.adjs[:i], m.adjs[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *policyAdjRepo) DeleteByWave(ctx context.Context, waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []domain.FulfillmentAdjustment
	for _, a := range m.adjs {
		if a.WaveID != waveID {
			kept = append(kept, a)
		}
	}
	m.adjs = kept
	return nil
}

type policyProductRepo struct {
	mu       sync.Mutex
	products map[uint]*domain.Product
}

func newPolicyProductRepo() *policyProductRepo {
	return &policyProductRepo{products: make(map[uint]*domain.Product)}
}

func (m *policyProductRepo) Create(ctx context.Context, product *domain.Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if product.ID == 0 {
		product.ID = uint(len(m.products) + 1)
	}
	cp := *product
	m.products[product.ID] = &cp
	return nil
}

func (m *policyProductRepo) FindByID(ctx context.Context, id uint) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[id]
	if !ok {
		return nil, fmt.Errorf("product %d not found", id)
	}
	cp := *p
	return &cp, nil
}

func (m *policyProductRepo) FindByWaveAndID(ctx context.Context, waveID uint, id uint) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[id]
	if !ok || p.WaveID != waveID {
		return nil, fmt.Errorf("product %d not found in wave %d", id, waveID)
	}
	cp := *p
	return &cp, nil
}

func (m *policyProductRepo) ListByWave(ctx context.Context, waveID uint) ([]domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Product
	for _, p := range m.products {
		if p.WaveID == waveID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (m *policyProductRepo) FindByWaveAndSKU(ctx context.Context, waveID uint, platform, sku string) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.products {
		if p.WaveID == waveID && p.SupplierPlatform == platform && p.FactorySKU == sku {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("product not found")
}

func (m *policyProductRepo) DeleteByWave(ctx context.Context, waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, p := range m.products {
		if p.WaveID == waveID {
			delete(m.products, id)
		}
	}
	return nil
}

// ── Tests ──

func TestReconcileWave_EmptyRules_ReturnsZeroCreated(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	// Add participants to the wave (they exist but no rules match them).
	waveRepo.participants[1] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
		{ID: 2, WaveID: 1, CustomerProfileID: 101, IdentityPlatform: "bilibili", GiftLevel: "L2"},
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)

	result, err := uc.ReconcileWave(context.Background(), 1)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}

	if result.Created != 0 {
		t.Errorf("expected 0 created lines (no rules), got %d", result.Created)
	}
	if result.ReplayedCount != 0 {
		t.Errorf("expected 0 replayed adjustments, got %d", result.ReplayedCount)
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(result.Failures))
	}
}

func TestReconcileWave_SingleRule_MatchesAll(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	waveID := uint(1)

	// Two participants
	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: waveID, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
		{ID: 2, WaveID: waveID, CustomerProfileID: 101, IdentityPlatform: "bilibili", GiftLevel: "L2"},
	}

	// One rule: wave_all selector, product 10, quantity 3
	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            10,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 3,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)

	result, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("expected 2 created lines (2 participants matched), got %d", result.Created)
	}

	// Verify persisted lines
	lines, _ := fulfillRepo.ListByWave(context.Background(), waveID)
	if len(lines) != 2 {
		t.Fatalf("expected 2 persisted lines, got %d", len(lines))
	}
	for i, fl := range lines {
		if fl.Quantity != 3 {
			t.Errorf("line %d: expected quantity 3, got %d", i, fl.Quantity)
		}
		if fl.GeneratedBy != "allocation_policy_driven" {
			t.Errorf("line %d: expected GeneratedBy='allocation_policy_driven', got %q", i, fl.GeneratedBy)
		}
		if fl.LineReason != "entitlement" {
			t.Errorf("line %d: expected LineReason='entitlement', got %q", i, fl.LineReason)
		}
		if fl.AllocationState != "ready" {
			t.Errorf("line %d: expected AllocationState='ready', got %q", i, fl.AllocationState)
		}
	}
}

func TestReconcileWave_Idempotent(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	waveID := uint(1)

	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: waveID, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
	}

	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            5,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 2,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)

	// First reconcile
	result1, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("first ReconcileWave failed: %v", err)
	}
	if result1.Created != 1 {
		t.Fatalf("first: expected 1 created, got %d", result1.Created)
	}

	// Second reconcile — should delete old lines and rebuild (idempotent)
	result2, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("second ReconcileWave failed: %v", err)
	}
	if result2.Created != 1 {
		t.Errorf("second: expected 1 created (idempotent rebuild), got %d", result2.Created)
	}

	// Only 1 line should exist (not 2)
	lines, _ := fulfillRepo.ListByWave(context.Background(), waveID)
	if len(lines) != 1 {
		t.Errorf("expected 1 line after idempotent rebuild, got %d", len(lines))
	}
}

func TestReconcileWave_InactiveRulesSkipped(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	waveID := uint(1)

	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: waveID, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
	}

	// Inactive rule — should be skipped
	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            5,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 10,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               false,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}

	productRepo := newPolicyProductRepo()
	productRepo.products[10] = &domain.Product{ID: 10, WaveID: 1, Name: "Wave1 Product"}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, productRepo)

	result, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}

	if result.Created != 0 {
		t.Errorf("expected 0 created (inactive rule), got %d", result.Created)
	}
}

func TestCreateRule_And_ListRulesByWave(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	productRepo := newPolicyProductRepo()
	productRepo.products[10] = &domain.Product{ID: 10, WaveID: 1, Name: "Wave1 Product"}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, productRepo)

	selectorJSON, _ := json.Marshal(domain.SelectorPayload{Type: "wave_all"})
	input := dto.CreateAllocationPolicyRuleInput{
		WaveID:               1,
		ProductID:            10,
		SelectorPayload:      selectorJSON,
		ProductTargetRef:     "product:10",
		ContributionQuantity: 5,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	}

	created, err := uc.CreateRule(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected rule ID to be set")
	}
	if created.WaveID != 1 {
		t.Errorf("expected WaveID=1, got %d", created.WaveID)
	}
	if created.ContributionQuantity != 5 {
		t.Errorf("expected ContributionQuantity=5, got %d", created.ContributionQuantity)
	}

	// List
	rules, err := uc.ListRulesByWave(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListRulesByWave failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != created.ID {
		t.Errorf("expected rule ID %d, got %d", created.ID, rules[0].ID)
	}
}

func TestUpdateRule(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	productRepo := newPolicyProductRepo()
	productRepo.products[10] = &domain.Product{ID: 10, WaveID: 1, Name: "Wave1 Product"}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, productRepo)

	// Create a rule first
	selectorJSON2, _ := json.Marshal(domain.SelectorPayload{Type: "wave_all"})
	created, err := uc.CreateRule(context.Background(), dto.CreateAllocationPolicyRuleInput{
		WaveID:               1,
		ProductID:            10,
		SelectorPayload:      selectorJSON2,
		ContributionQuantity: 5,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	})
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	// Update quantity
	newQty := 8
	updated, err := uc.UpdateRule(context.Background(), dto.UpdateAllocationPolicyRuleInput{
		ID:                   created.ID,
		ContributionQuantity: &newQty,
	})
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}
	if updated.ContributionQuantity != 8 {
		t.Errorf("expected updated quantity=8, got %d", updated.ContributionQuantity)
	}
	// Other fields unchanged
	if updated.WaveID != 1 {
		t.Errorf("expected WaveID unchanged=1, got %d", updated.WaveID)
	}
}

func TestDeleteRule(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)

	selectorJSON3, _ := json.Marshal(domain.SelectorPayload{Type: "wave_all"})
	created, err := uc.CreateRule(context.Background(), dto.CreateAllocationPolicyRuleInput{
		WaveID:               1,
		ProductID:            10,
		SelectorPayload:      selectorJSON3,
		ContributionQuantity: 5,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	})
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	err = uc.DeleteRule(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	rules, _ := uc.ListRulesByWave(context.Background(), 1)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestCreateRuleRejectsProductFromDifferentWave(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()
	productRepo := newPolicyProductRepo()
	productRepo.products[10] = &domain.Product{ID: 10, WaveID: 2, Name: "Wave2 Product"}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, productRepo)

	selectorJSON4, _ := json.Marshal(domain.SelectorPayload{Type: "wave_all"})
	_, err := uc.CreateRule(context.Background(), dto.CreateAllocationPolicyRuleInput{
		WaveID:               1,
		ProductID:            10,
		SelectorPayload:      selectorJSON4,
		ContributionQuantity: 1,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	})
	if err == nil {
		t.Fatal("expected error for cross-wave product, got nil")
	}
}

func TestUpdateRuleRejectsProductFromDifferentWave(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()
	productRepo := newPolicyProductRepo()
	productRepo.products[10] = &domain.Product{ID: 10, WaveID: 1, Name: "Wave1 Product"}
	productRepo.products[20] = &domain.Product{ID: 20, WaveID: 2, Name: "Wave2 Product"}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, productRepo)

	selectorJSON5, _ := json.Marshal(domain.SelectorPayload{Type: "wave_all"})
	created, err := uc.CreateRule(context.Background(), dto.CreateAllocationPolicyRuleInput{
		WaveID:               1,
		ProductID:            10,
		SelectorPayload:      selectorJSON5,
		ContributionQuantity: 1,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	})
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	newProductID := uint(20)
	_, err = uc.UpdateRule(context.Background(), dto.UpdateAllocationPolicyRuleInput{
		ID:        created.ID,
		ProductID: &newProductID,
	})
	if err == nil {
		t.Fatal("expected error for cross-wave product update, got nil")
	}
}

func TestReconcileWave_ReanchorsFulfillmentLineAdjustmentTargetAfterRebuild(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	waveID := uint(1)
	participantID := uint(10)
	customerProfileID := uint(100)
	productID := uint(55)

	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: participantID, WaveID: waveID, CustomerProfileID: customerProfileID, IdentityPlatform: "bilibili", GiftLevel: "L1"},
	}

	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            productID,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 2,
		RuleKind:             "entitlement",
		Priority:             1,
		Active:               true,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)

	first, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("first ReconcileWave failed: %v", err)
	}
	if first.Created != 1 {
		t.Fatalf("expected first reconcile to create 1 line, got %d", first.Created)
	}

	lines, _ := fulfillRepo.ListByWave(context.Background(), waveID)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line after first reconcile, got %d", len(lines))
	}
	originalLineID := lines[0].ID

	if err := adjRepo.Create(context.Background(), &domain.FulfillmentAdjustment{
		WaveID:            waveID,
		TargetKind:        "fulfillment_line",
		FulfillmentLineID: &originalLineID,
		AdjustmentKind:    "add",
		QuantityDelta:     3,
		OperatorID:        "op-1",
	}); err != nil {
		t.Fatalf("setup adjustment Create failed: %v", err)
	}

	second, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("second ReconcileWave failed: %v", err)
	}
	if len(second.Failures) != 0 {
		t.Fatalf("expected 0 replay failures after re-anchor, got %v", second.Failures)
	}

	updatedAdj, err := adjRepo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByID(updated adjustment): %v", err)
	}
	if updatedAdj == nil || updatedAdj.FulfillmentLineID == nil {
		t.Fatal("expected updated adjustment with fulfillment line target")
	}
	if *updatedAdj.FulfillmentLineID == originalLineID {
		t.Fatalf("expected adjustment target to be re-anchored away from old line ID %d", originalLineID)
	}

	lines, _ = fulfillRepo.ListByWave(context.Background(), waveID)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line after second reconcile, got %d", len(lines))
	}
	if lines[0].Quantity != 5 {
		t.Fatalf("expected replayed quantity 5 after rebuild + add, got %d", lines[0].Quantity)
	}
	if *updatedAdj.FulfillmentLineID != lines[0].ID {
		t.Fatalf("expected adjustment target %d to match current line ID %d", *updatedAdj.FulfillmentLineID, lines[0].ID)
	}
}
