package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock DocumentTemplateRepository ──

type mockDocumentTemplateRepo struct {
	mu      sync.Mutex
	records map[uint]*domain.DocumentTemplate
	byKey   map[string]*domain.DocumentTemplate
	lastID  uint
	failOn  string // "create" to simulate Create error
}

func newMockDocumentTemplateRepo() *mockDocumentTemplateRepo {
	return &mockDocumentTemplateRepo{
		records: make(map[uint]*domain.DocumentTemplate),
		byKey:   make(map[string]*domain.DocumentTemplate),
	}
}

func (m *mockDocumentTemplateRepo) Create(t *domain.DocumentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn == "create" {
		return fmt.Errorf("mock: create failed")
	}
	m.lastID++
	t.ID = m.lastID
	t.CreatedAt = "2024-01-01T00:00:00Z"
	t.UpdatedAt = "2024-01-01T00:00:00Z"
	cp := *t
	m.records[t.ID] = &cp
	m.byKey[t.TemplateKey] = &cp
	return nil
}

func (m *mockDocumentTemplateRepo) FindByID(id uint) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *mockDocumentTemplateRepo) FindByKey(key string) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.byKey[key]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *mockDocumentTemplateRepo) List() ([]domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.DocumentTemplate, 0, len(m.records))
	for _, t := range m.records {
		out = append(out, *t)
	}
	return out, nil
}

func (m *mockDocumentTemplateRepo) ListByDocumentType(docType string) ([]domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.DocumentTemplate
	for _, t := range m.records {
		if t.DocumentType == docType {
			out = append(out, *t)
		}
	}
	return out, nil
}

// ── mock ProfileTemplateBindingRepository ──

type mockProfileTemplateBindingRepo struct {
	mu      sync.Mutex
	records map[uint]*domain.IntegrationProfileTemplateBinding
	lastID  uint
}

func newMockProfileTemplateBindingRepo() *mockProfileTemplateBindingRepo {
	return &mockProfileTemplateBindingRepo{
		records: make(map[uint]*domain.IntegrationProfileTemplateBinding),
	}
}

func (m *mockProfileTemplateBindingRepo) Create(b *domain.IntegrationProfileTemplateBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	b.ID = m.lastID
	b.CreatedAt = "2024-01-01T00:00:00Z"
	cp := *b
	m.records[b.ID] = &cp
	return nil
}

func (m *mockProfileTemplateBindingRepo) ListByProfile(profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.IntegrationProfileTemplateBinding
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (m *mockProfileTemplateBindingRepo) FindDefaultByProfileAndType(profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID && b.DocumentType == docType && b.IsDefault {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockProfileTemplateBindingRepo) Delete(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, id)
	return nil
}

// ── mock IntegrationProfileRepository (template tests) ──

type mockIntegrationProfileRepoForTemplate struct {
	mu       sync.Mutex
	profiles map[uint]*domain.IntegrationProfile
}

func newMockIntegrationProfileRepoForTemplate() *mockIntegrationProfileRepoForTemplate {
	return &mockIntegrationProfileRepoForTemplate{
		profiles: make(map[uint]*domain.IntegrationProfile),
	}
}

func (m *mockIntegrationProfileRepoForTemplate) Create(p *domain.IntegrationProfile) error {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) FindByID(id uint) (*domain.IntegrationProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *mockIntegrationProfileRepoForTemplate) FindByProfileKey(key string) (*domain.IntegrationProfile, error) {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) List() ([]domain.IntegrationProfile, error) {
	panic("not implemented")
}

// ── test setup ──

type templateTestSetup struct {
	templateRepo *mockDocumentTemplateRepo
	bindingRepo  *mockProfileTemplateBindingRepo
	profileRepo  *mockIntegrationProfileRepoForTemplate
	uc           TemplateManagementUseCase
}

func newTemplateTestSetup() *templateTestSetup {
	tr := newMockDocumentTemplateRepo()
	br := newMockProfileTemplateBindingRepo()
	pr := newMockIntegrationProfileRepoForTemplate()
	// Seed a profile so binding tests can validate it.
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, ProfileKey: "test-profile"}
	return &templateTestSetup{
		templateRepo: tr,
		bindingRepo:  br,
		profileRepo:  pr,
		uc:           NewTemplateManagementUseCase(tr, br, pr),
	}
}

func validCreateTemplateInput() dto.CreateDocumentTemplateInput {
	return dto.CreateDocumentTemplateInput{
		TemplateKey:  "tmpl-001",
		DocumentType: "import_entitlement",
		Format:       "csv",
		MappingRules: `{"col":"value"}`,
		ExtraData:    "",
	}
}

// ── tests ──

func TestCreateDocumentTemplateSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	result, err := s.uc.CreateDocumentTemplate(validCreateTemplateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if result.TemplateKey != "tmpl-001" {
		t.Errorf("TemplateKey = %q, want tmpl-001", result.TemplateKey)
	}
	if result.DocumentType != "import_entitlement" {
		t.Errorf("DocumentType = %q, want import_entitlement", result.DocumentType)
	}
	if result.Format != "csv" {
		t.Errorf("Format = %q, want csv", result.Format)
	}
}

func TestCreateDocumentTemplateInvalidType(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	input := validCreateTemplateInput()
	input.DocumentType = "not_a_real_type"

	_, err := s.uc.CreateDocumentTemplate(input)
	if err == nil {
		t.Fatal("expected error for invalid documentType, got nil")
	}
}

func TestCreateDocumentTemplateInvalidFormat(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	input := validCreateTemplateInput()
	input.Format = "pdf"

	_, err := s.uc.CreateDocumentTemplate(input)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestBindTemplateToProfileSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	// Create a template first.
	tmpl, err := s.uc.CreateDocumentTemplate(validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}

	bindInput := dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	}
	binding, err := s.uc.BindTemplateToProfile(bindInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if binding.ID == 0 {
		t.Error("expected non-zero binding ID")
	}
	if binding.TemplateID != tmpl.ID {
		t.Errorf("TemplateID = %d, want %d", binding.TemplateID, tmpl.ID)
	}
	if !binding.IsDefault {
		t.Error("expected IsDefault = true")
	}
}

func TestGetDefaultTemplateForProfile(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	// Create template and bind it as default.
	tmpl, err := s.uc.CreateDocumentTemplate(validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}
	_, err = s.uc.BindTemplateToProfile(dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	})
	if err != nil {
		t.Fatalf("setup: bind template: %v", err)
	}

	result, err := s.uc.GetDefaultTemplateForProfile(1, "import_entitlement")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected a template, got nil")
	}
	if result.ID != tmpl.ID {
		t.Errorf("template ID = %d, want %d", result.ID, tmpl.ID)
	}
	if result.TemplateKey != "tmpl-001" {
		t.Errorf("TemplateKey = %q, want tmpl-001", result.TemplateKey)
	}
}
