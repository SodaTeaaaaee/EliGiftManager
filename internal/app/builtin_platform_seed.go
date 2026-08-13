package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// Builtin platform install keys used by SeedBuiltinPlatform / the integrations page.
// These are not ProfileKey values: each key maps onto one existing demo seed.
const (
	BuiltinPlatformBilibili = "bilibili"
	BuiltinPlatformRouzao   = "rouzao"
)

// NormalizeBuiltinPlatformKey accepts the public install keys plus the seeded
// profile keys so callers can pass either form. membership_default is not a
// builtin platform and is rejected.
func NormalizeBuiltinPlatformKey(key string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case BuiltinPlatformBilibili, BilibiliDemoProfileKey:
		return BuiltinPlatformBilibili, nil
	case BuiltinPlatformRouzao, CatalogDemoProfileKey:
		return BuiltinPlatformRouzao, nil
	default:
		return "", fmt.Errorf("unknown builtin platform %q (want %q or %q)", key, BuiltinPlatformBilibili, BuiltinPlatformRouzao)
	}
}

// SeedBuiltinPlatform installs one named builtin (Bilibili source or 柔造 factory)
// by reusing SeedBilibiliDemo / SeedCatalogDemo, including that demo's default
// template bindings. Idempotent if already installed: existing operator
// mappings and profile capabilities are not overwritten. Does not seed
// membership_default.
func SeedBuiltinPlatform(
	ctx context.Context,
	key string,
	profileRepo domain.IntegrationProfileRepository,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
) (*dto.IntegrationProfileDTO, error) {
	normalized, err := NormalizeBuiltinPlatformKey(key)
	if err != nil {
		return nil, err
	}

	var profile *domain.IntegrationProfile
	switch normalized {
	case BuiltinPlatformBilibili:
		profile, err = SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
	case BuiltinPlatformRouzao:
		profile, err = SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo)
	default:
		return nil, fmt.Errorf("unknown builtin platform %q", key)
	}
	if err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}
