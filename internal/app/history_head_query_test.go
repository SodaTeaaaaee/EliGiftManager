package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock HistoryScopeRepository ──

type mockHistoryScopeRepo struct {
	mu     sync.Mutex
	scopes map[string]*domain.HistoryScope // key: "scopeType:scopeKey"
	lastID uint
	errOn  string // method name to fail on
}

func newMockHistoryScopeRepo() *mockHistoryScopeRepo {
	return &mockHistoryScopeRepo{scopes: make(map[string]*domain.HistoryScope)}
}

func (m *mockHistoryScopeRepo) key(scopeType, scopeKey string) string {
	return scopeType + ":" + scopeKey
}

func (m *mockHistoryScopeRepo) Create(scope *domain.HistoryScope) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	scope.ID = m.lastID
	cp := *scope
	m.scopes[m.key(scope.ScopeType, scope.ScopeKey)] = &cp
	return nil
}

func (m *mockHistoryScopeRepo) FindByID(id uint) (*domain.HistoryScope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.scopes {
		if s.ID == id {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockHistoryScopeRepo) FindByScopeTypeAndKey(scopeType, scopeKey string) (*domain.HistoryScope, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.errOn == "FindByScopeTypeAndKey" {
		return nil, fmt.Errorf("mock: FindByScopeTypeAndKey failed")
	}
	s, ok := m.scopes[m.key(scopeType, scopeKey)]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (m *mockHistoryScopeRepo) UpdateHead(scopeID uint, headNodeID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.scopes {
		if s.ID == scopeID {
			s.CurrentHeadNodeID = headNodeID
			return nil
		}
	}
	return fmt.Errorf("scope %d not found", scopeID)
}

func (m *mockHistoryScopeRepo) FindOrCreate(scopeType string, scopeKey string) (*domain.HistoryScope, error) {
	existing, err := m.FindByScopeTypeAndKey(scopeType, scopeKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	scope := &domain.HistoryScope{ScopeType: scopeType, ScopeKey: scopeKey}
	if err := m.Create(scope); err != nil {
		return nil, err
	}
	return scope, nil
}

// ── mock HistoryNodeRepository ──

type mockHistoryNodeRepo struct {
	mu     sync.Mutex
	nodes  map[uint]*domain.HistoryNode
	lastID uint
	errOn  string // method name to fail on
}

func newMockHistoryNodeRepo() *mockHistoryNodeRepo {
	return &mockHistoryNodeRepo{nodes: make(map[uint]*domain.HistoryNode)}
}

func (m *mockHistoryNodeRepo) Create(node *domain.HistoryNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	node.ID = m.lastID
	cp := *node
	m.nodes[node.ID] = &cp
	return nil
}

func (m *mockHistoryNodeRepo) FindByID(id uint) (*domain.HistoryNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.errOn == "FindByID" {
		return nil, fmt.Errorf("mock: FindByID failed")
	}
	n, ok := m.nodes[id]
	if !ok {
		return nil, nil
	}
	cp := *n
	return &cp, nil
}

func (m *mockHistoryNodeRepo) UpdatePreferredRedoChild(nodeID uint, childID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d not found", nodeID)
	}
	n.PreferredRedoChildID = childID
	return nil
}

func (m *mockHistoryNodeRepo) ListByScopeRecent(scopeID uint, limit int) ([]domain.HistoryNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.HistoryNode
	for _, n := range m.nodes {
		if n.HistoryScopeID == scopeID {
			result = append(result, *n)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ── tests: HistoryHeadQueryUseCase ──

// When no HistoryScope exists for a wave, GetCurrentProjectionHash must return ("", nil).
// This is the steady-state for all waves before Stage F history recording is implemented.
func TestHistoryHeadQueryNoScopeReturnsEmptyHash(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	uc := NewHistoryHeadQueryUseCase(scopeRepo, nodeRepo)

	hash, err := uc.GetCurrentProjectionHash(42)
	if err != nil {
		t.Fatalf("expected nil error when no scope exists, got: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash when no scope exists, got %q", hash)
	}
}

// When no HistoryScope exists, GetCurrentHeadNodeIDAndHash must return (0, "", nil).
func TestHistoryHeadQueryNoScopeReturnsZeroNodeID(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	uc := NewHistoryHeadQueryUseCase(scopeRepo, nodeRepo)

	nodeID, hash, err := uc.GetCurrentHeadNodeIDAndHash(42)
	if err != nil {
		t.Fatalf("expected nil error when no scope exists, got: %v", err)
	}
	if nodeID != 0 {
		t.Errorf("expected nodeID=0 when no scope exists, got %d", nodeID)
	}
	if hash != "" {
		t.Errorf("expected empty hash when no scope exists, got %q", hash)
	}
}

// When a scope exists but CurrentHeadNodeID == 0 (no nodes ever recorded),
// GetCurrentProjectionHash must return ("", nil) — not an error.
func TestHistoryHeadQueryScopeExistsButNoHeadNode(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()

	// Create a scope with no head node
	if err := scopeRepo.Create(&domain.HistoryScope{
		ScopeType:         "wave",
		ScopeKey:          "7",
		CurrentHeadNodeID: 0,
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	uc := NewHistoryHeadQueryUseCase(scopeRepo, nodeRepo)

	hash, err := uc.GetCurrentProjectionHash(7)
	if err != nil {
		t.Fatalf("expected nil error when scope has no head node, got: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash when scope has no head node, got %q", hash)
	}
}

// When a scope points to a head node, GetCurrentProjectionHash returns that node's hash.
func TestHistoryHeadQueryReturnsNodeHash(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()

	node := &domain.HistoryNode{
		ProjectionHash: "abc123",
		CommandSummary: "test command",
	}
	if err := nodeRepo.Create(node); err != nil {
		t.Fatalf("setup node: %v", err)
	}

	scope := &domain.HistoryScope{
		ScopeType:         "wave",
		ScopeKey:          "5",
		CurrentHeadNodeID: node.ID,
	}
	if err := scopeRepo.Create(scope); err != nil {
		t.Fatalf("setup scope: %v", err)
	}

	uc := NewHistoryHeadQueryUseCase(scopeRepo, nodeRepo)

	hash, err := uc.GetCurrentProjectionHash(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("expected hash %q, got %q", "abc123", hash)
	}
}

// When scope repo returns an error, the error must propagate.
func TestHistoryHeadQueryScopeRepoErrorPropagates(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	scopeRepo.errOn = "FindByScopeTypeAndKey"
	nodeRepo := newMockHistoryNodeRepo()
	uc := NewHistoryHeadQueryUseCase(scopeRepo, nodeRepo)

	_, err := uc.GetCurrentProjectionHash(1)
	if err == nil {
		t.Fatal("expected error when scope repo fails, got nil")
	}
}

// ── tests: UndoRedoUseCase with no history ──

// Undo on a wave with no history scope must return a clear error, not panic or return empty string.
func TestUndoNoHistoryScopeReturnsError(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo)

	_, err := uc.Undo(99)
	if err == nil {
		t.Fatal("expected error when no history scope exists for wave, got nil")
	}
}

// Redo on a wave with no history scope must return a clear error.
func TestRedoNoHistoryScopeReturnsError(t *testing.T) {
	t.Parallel()

	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo)

	_, err := uc.Redo(99)
	if err == nil {
		t.Fatal("expected error when no history scope exists for wave, got nil")
	}
}
