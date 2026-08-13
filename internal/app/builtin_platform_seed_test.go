package app

import (
	"context"
	"strings"
	"testing"
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

	if _, err := profileRepo.FindByProfileKey(ctx, "membership_default"); err == nil {
		t.Fatal("membership_default must not be created by per-platform install")
	}
	if _, err := profileRepo.FindByProfileKey(ctx, BilibiliDemoProfileKey); err == nil {
		t.Fatal("bilibili must not be created when installing rouzao")
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
	second, err := SeedBuiltinPlatform(ctx, "Bilibili", profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.ID != second.ID || first.ProfileKey != second.ProfileKey {
		t.Fatalf("idempotent mismatch: first=%+v second=%+v", first, second)
	}

	listed, err := profileRepo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("profiles = %d, want 1 after repeated install", len(listed))
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
