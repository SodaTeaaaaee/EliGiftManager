package app

import (
	"context"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestSeedBuiltinPlatform_Bilibili(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	dtoOut, err := SeedBuiltinPlatform(ctx, "bilibili", profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("SeedBuiltinPlatform(bilibili): %v", err)
	}
	if dtoOut == nil || dtoOut.ProfileKey != BilibiliDemoProfileKey {
		t.Fatalf("profile = %+v, want key %q", dtoOut, BilibiliDemoProfileKey)
	}

	assertBilibiliDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, dtoOut.ID)

	if _, err := profileRepo.FindByProfileKey(ctx, "membership_default"); err == nil {
		t.Fatal("membership_default must not be created by per-platform install")
	}
	if _, err := profileRepo.FindByProfileKey(ctx, CatalogDemoProfileKey); err == nil {
		t.Fatal("rouzao must not be created when installing bilibili")
	}
}

func TestSeedBuiltinPlatform_Rouzao(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	dtoOut, err := SeedBuiltinPlatform(ctx, "rouzao", profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("SeedBuiltinPlatform(rouzao): %v", err)
	}
	if dtoOut == nil || dtoOut.ProfileKey != CatalogDemoProfileKey {
		t.Fatalf("profile = %+v, want key %q", dtoOut, CatalogDemoProfileKey)
	}
	if dtoOut.DemandKind != "" {
		t.Errorf("rouzao leftover DemandKind = %q, want empty", dtoOut.DemandKind)
	}

	assertRouzaoDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, dtoOut.ID)

	if _, err := profileRepo.FindByProfileKey(ctx, "membership_default"); err == nil {
		t.Fatal("membership_default must not be created by per-platform install")
	}
	if _, err := profileRepo.FindByProfileKey(ctx, BilibiliDemoProfileKey); err == nil {
		t.Fatal("bilibili must not be created when installing rouzao")
	}
}

func TestSeedBuiltinPlatform_BothPlatformsDefaultBindings(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	bili, err := SeedBuiltinPlatform(ctx, BuiltinPlatformBilibili, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("bilibili: %v", err)
	}
	rouzao, err := SeedBuiltinPlatform(ctx, BuiltinPlatformRouzao, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("rouzao: %v", err)
	}
	if bili.ID == rouzao.ID {
		t.Fatalf("platforms must not share a profile ID: %d", bili.ID)
	}

	assertBilibiliDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, bili.ID)
	assertRouzaoDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, rouzao.ID)

	listed, err := profileRepo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("profiles = %d, want 2 after installing both builtins", len(listed))
	}
}

func TestSeedBuiltinPlatform_IdempotentAndAcceptsProfileKey(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	first, err := SeedBuiltinPlatform(ctx, BilibiliDemoProfileKey, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	assertBilibiliDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, first.ID)

	second, err := SeedBuiltinPlatform(ctx, "Bilibili", profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.ID != second.ID || first.ProfileKey != second.ProfileKey {
		t.Fatalf("idempotent mismatch: first=%+v second=%+v", first, second)
	}
	assertBilibiliDefaultBindings(t, ctx, profileRepo, templateRepo, bindingRepo, second.ID)

	listed, err := profileRepo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("profiles = %d, want 1 after repeated install", len(listed))
	}
	bindings, err := bindingRepo.ListByProfile(ctx, first.ID)
	if err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if len(bindings) != 4 {
		t.Fatalf("bilibili bindings = %d, want 4 after repeated install", len(bindings))
	}
}

func TestSeedBuiltinPlatform_RejectsUnknownAndMembershipDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	profileRepo := newMockIntegrationProfileRepoSimple()

	_, err := SeedBuiltinPlatform(ctx, "membership_default", profileRepo, nil, nil)
	if err == nil {
		t.Fatal("expected error for membership_default")
	}
	if !strings.Contains(err.Error(), "unknown builtin platform") {
		t.Fatalf("error = %v, want unknown builtin platform", err)
	}

	_, err = SeedBuiltinPlatform(ctx, "patreon", profileRepo, nil, nil)
	if err == nil {
		t.Fatal("expected error for patreon")
	}
}

func assertBilibiliDefaultBindings(
	t *testing.T,
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID uint,
) {
	t.Helper()
	profile := mustFindProfile(t, ctx, profileRepo, profileID)
	for _, check := range []struct {
		docType     string
		templateKey string
	}{
		{BilibiliImportEntitlementDocType, BilibiliImportEntitlementTemplateKey},
		{BilibiliImportSalesOrderDocType, BilibiliImportSalesOrderTemplateKey},
		{BilibiliExportTrackingDocType, BilibiliExportTrackingTemplateKey},
		{BilibiliImportCarrierDocType, BilibiliImportCarrierTemplateKey},
	} {
		assertDefaultBinding(t, ctx, profile, templateRepo, bindingRepo, check.docType, check.templateKey)
	}
}

func assertRouzaoDefaultBindings(
	t *testing.T,
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID uint,
) {
	t.Helper()
	profile := mustFindProfile(t, ctx, profileRepo, profileID)
	if profile.DemandKind != "" {
		t.Errorf("factory leftover DemandKind = %q, want empty", profile.DemandKind)
	}
	for _, check := range []struct {
		docType     string
		templateKey string
	}{
		{CatalogDemoDocType, CatalogDemoTemplateKey},
		{ShipmentDemoDocType, ShipmentDemoTemplateKey},
		{SupplierOrderDemoDocType, SupplierOrderDemoTemplateKey},
	} {
		assertDefaultBinding(t, ctx, profile, templateRepo, bindingRepo, check.docType, check.templateKey)
	}
}

func mustFindProfile(t *testing.T, ctx context.Context, repo domain.IntegrationProfileRepository, id uint) *domain.IntegrationProfile {
	t.Helper()
	profile, err := repo.FindByID(ctx, id)
	if err != nil || profile == nil {
		t.Fatalf("FindByID(%d): profile=%v err=%v", id, profile, err)
	}
	return profile
}

func assertDefaultBinding(
	t *testing.T,
	ctx context.Context,
	profile *domain.IntegrationProfile,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	docType, templateKey string,
) {
	t.Helper()
	if err := ValidateProfileDocumentType(profile, docType); err != nil {
		t.Fatalf("ValidateProfileDocumentType(%q) on %s: %v", docType, profile.ProfileKey, err)
	}
	tmpl, err := templateRepo.FindByKey(ctx, templateKey)
	if err != nil || tmpl == nil {
		t.Fatalf("template %q: tmpl=%v err=%v", templateKey, tmpl, err)
	}
	if tmpl.DocumentType != docType {
		t.Fatalf("template %q DocumentType = %q, want %q", templateKey, tmpl.DocumentType, docType)
	}
	binding, err := bindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, docType)
	if err != nil || binding == nil {
		t.Fatalf("default binding profile=%d type=%s: binding=%v err=%v", profile.ID, docType, binding, err)
	}
	if binding.TemplateID != tmpl.ID || !binding.IsDefault {
		t.Fatalf("default binding %+v, want templateID=%d isDefault", binding, tmpl.ID)
	}
}
