package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

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

func (m *mockDocumentTemplateRepo) Create(ctx context.Context, t *domain.DocumentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn == "create" {
		return fmt.Errorf("mock: create failed")
	}
	m.lastID++
	t.ID = m.lastID
	t.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t.UpdatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cp := *t
	m.records[t.ID] = &cp
	m.byKey[t.TemplateKey] = &cp
	return nil
}

func (m *mockDocumentTemplateRepo) FindByID(ctx context.Context, id uint) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *mockDocumentTemplateRepo) FindByKey(ctx context.Context, key string) (*domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.byKey[key]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *mockDocumentTemplateRepo) List(ctx context.Context) ([]domain.DocumentTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.DocumentTemplate, 0, len(m.records))
	for _, t := range m.records {
		out = append(out, *t)
	}
	return out, nil
}

func (m *mockDocumentTemplateRepo) ListByDocumentType(ctx context.Context, docType string) ([]domain.DocumentTemplate, error) {
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

func (m *mockDocumentTemplateRepo) Update(ctx context.Context, t *domain.DocumentTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.records[t.ID]
	if !ok {
		return fmt.Errorf("mock: template %d not found", t.ID)
	}
	// Keep key/type immutable in the mock the same way the use case enforces.
	t.TemplateKey = existing.TemplateKey
	t.DocumentType = existing.DocumentType
	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	cp := *t
	m.records[t.ID] = &cp
	m.byKey[t.TemplateKey] = &cp
	return nil
}

func (m *mockDocumentTemplateRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.records[id]
	if !ok {
		return fmt.Errorf("mock: template %d not found", id)
	}
	delete(m.byKey, existing.TemplateKey)
	delete(m.records, id)
	return nil
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

func (m *mockProfileTemplateBindingRepo) Create(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	b.ID = m.lastID
	b.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cp := *b
	m.records[b.ID] = &cp
	return nil
}

func (m *mockProfileTemplateBindingRepo) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	cp := *b
	return &cp, nil
}

func (m *mockProfileTemplateBindingRepo) ListByProfile(ctx context.Context, profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
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

func (m *mockProfileTemplateBindingRepo) ListByTemplateID(ctx context.Context, templateID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.IntegrationProfileTemplateBinding
	for _, b := range m.records {
		if b.TemplateID == templateID {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (m *mockProfileTemplateBindingRepo) FindDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
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

func (m *mockProfileTemplateBindingRepo) ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID && b.DocumentType == docType && b.IsDefault {
			b.IsDefault = false
		}
	}
	return nil
}

func (m *mockProfileTemplateBindingRepo) Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.records[b.ID]; !ok {
		return fmt.Errorf("mock: binding %d not found", b.ID)
	}
	cp := *b
	m.records[b.ID] = &cp
	return nil
}

func (m *mockProfileTemplateBindingRepo) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, id)
	return nil
}

func (m *mockProfileTemplateBindingRepo) CountByProfileID(ctx context.Context, profileID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, b := range m.records {
		if b.IntegrationProfileID == profileID {
			count++
		}
	}
	return count, nil
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

func (m *mockIntegrationProfileRepoForTemplate) Create(ctx context.Context, p *domain.IntegrationProfile) error {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *mockIntegrationProfileRepoForTemplate) FindByProfileKey(ctx context.Context, key string) (*domain.IntegrationProfile, error) {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) List(ctx context.Context) ([]domain.IntegrationProfile, error) {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) Update(ctx context.Context, profile *domain.IntegrationProfile) error {
	panic("not implemented")
}

func (m *mockIntegrationProfileRepoForTemplate) Delete(ctx context.Context, id uint) error {
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
		MappingRules: `{"columns":{"external_title":"Name","requested_quantity":"Qty"},"defaults":{"line_type":"sku_order"}}`,
		ExtraData:    "",
	}
}

// ── tests ──

func TestCreateDocumentTemplateSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	result, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
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

	_, err := s.uc.CreateDocumentTemplate(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid documentType, got nil")
	}
}

func TestCreateDocumentTemplateInvalidFormat(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	input := validCreateTemplateInput()
	input.Format = "pdf"

	_, err := s.uc.CreateDocumentTemplate(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestBindTemplateToProfileSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	// Create a template first.
	tmpl, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}

	bindInput := dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	}
	binding, err := s.uc.BindTemplateToProfile(context.Background(), bindInput)
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
	tmpl, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}
	_, err = s.uc.BindTemplateToProfile(context.Background(), dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	})
	if err != nil {
		t.Fatalf("setup: bind template: %v", err)
	}

	result, err := s.uc.GetDefaultTemplateForProfile(context.Background(), 1, "import_entitlement")
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

func TestUpdateDocumentTemplateSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	created, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}

	updated, err := s.uc.UpdateDocumentTemplate(context.Background(), dto.UpdateDocumentTemplateInput{
		ID:           created.ID,
		Format:       "xlsx",
		MappingRules: `{"columns":{"external_title":"Title","requested_quantity":"Amount"},"defaults":{"line_type":"sku_order"}}`,
		ExtraData:    `{"note":"updated"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Format != "xlsx" {
		t.Errorf("Format = %q, want xlsx", updated.Format)
	}
	if updated.TemplateKey != created.TemplateKey {
		t.Errorf("TemplateKey changed to %q, must stay %q", updated.TemplateKey, created.TemplateKey)
	}
	if updated.DocumentType != created.DocumentType {
		t.Errorf("DocumentType changed to %q, must stay %q", updated.DocumentType, created.DocumentType)
	}
	if updated.ExtraData != `{"note":"updated"}` {
		t.Errorf("ExtraData = %q, want updated note", updated.ExtraData)
	}
}

func TestUpdateDocumentTemplateInvalidFormat(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	created, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}

	_, err = s.uc.UpdateDocumentTemplate(context.Background(), dto.UpdateDocumentTemplateInput{
		ID:     created.ID,
		Format: "pdf",
	})
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestDeleteDocumentTemplateSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	created, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}

	if err := s.uc.DeleteDocumentTemplate(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.templateRepo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("lookup after delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected template to be gone after delete")
	}
}

func TestDeleteDocumentTemplateRejectsWhenBound(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	created, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}
	_, err = s.uc.BindTemplateToProfile(context.Background(), dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           created.ID,
		IsDefault:            true,
	})
	if err != nil {
		t.Fatalf("setup: bind template: %v", err)
	}

	err = s.uc.DeleteDocumentTemplate(context.Background(), created.ID)
	if err == nil {
		t.Fatal("expected delete to fail while bindings exist")
	}
	if !strings.Contains(err.Error(), "still referenced") {
		t.Errorf("error = %q, want mention of still referenced", err.Error())
	}
}

func TestUnbindTemplateSuccess(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	tmpl, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template: %v", err)
	}
	binding, err := s.uc.BindTemplateToProfile(context.Background(), dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            false,
	})
	if err != nil {
		t.Fatalf("setup: bind template: %v", err)
	}

	if err := s.uc.UnbindTemplate(context.Background(), binding.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := s.uc.ListBindingsByProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("list after unbind: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 bindings after unbind, got %d", len(list))
	}
}

func TestSetDefaultBindingSwitchesDefault(t *testing.T) {
	t.Parallel()
	s := newTemplateTestSetup()

	tmplA, err := s.uc.CreateDocumentTemplate(context.Background(), validCreateTemplateInput())
	if err != nil {
		t.Fatalf("setup: create template A: %v", err)
	}
	inputB := validCreateTemplateInput()
	inputB.TemplateKey = "tmpl-002"
	tmplB, err := s.uc.CreateDocumentTemplate(context.Background(), inputB)
	if err != nil {
		t.Fatalf("setup: create template B: %v", err)
	}

	bindingA, err := s.uc.BindTemplateToProfile(context.Background(), dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmplA.ID,
		IsDefault:            true,
	})
	if err != nil {
		t.Fatalf("setup: bind A: %v", err)
	}
	bindingB, err := s.uc.BindTemplateToProfile(context.Background(), dto.BindTemplateToProfileInput{
		IntegrationProfileID: 1,
		DocumentType:         "import_entitlement",
		TemplateID:           tmplB.ID,
		IsDefault:            false,
	})
	if err != nil {
		t.Fatalf("setup: bind B: %v", err)
	}

	if err := s.uc.SetDefaultBinding(context.Background(), bindingB.ID); err != nil {
		t.Fatalf("SetDefaultBinding: %v", err)
	}

	defaultTmpl, err := s.uc.GetDefaultTemplateForProfile(context.Background(), 1, "import_entitlement")
	if err != nil {
		t.Fatalf("GetDefaultTemplateForProfile: %v", err)
	}
	if defaultTmpl == nil || defaultTmpl.ID != tmplB.ID {
		t.Fatalf("expected default template %d, got %+v", tmplB.ID, defaultTmpl)
	}

	// Old default must no longer be default.
	gotA, err := s.bindingRepo.FindByID(context.Background(), bindingA.ID)
	if err != nil {
		t.Fatalf("lookup binding A: %v", err)
	}
	if gotA == nil || gotA.IsDefault {
		t.Fatalf("expected binding A IsDefault=false, got %+v", gotA)
	}
}
