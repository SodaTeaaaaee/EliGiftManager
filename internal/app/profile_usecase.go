package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- ProfileManagementUseCase ----

type profileManagementUseCase struct {
	repo                domain.IntegrationProfileRepository
	demandRepo          domain.DemandDocumentRepository
	channelSyncRepo     domain.ChannelSyncRepository
	templateBindingRepo domain.ProfileTemplateBindingRepository
	closureDecisionRepo domain.ChannelClosureDecisionRepository
	executorProvider    ExecutorProvider
	referenceRepo       domain.IntegrationProfileReferenceRepository
}

// WithIntegrationProfileReferenceRepo enables deletion guards for references
// outside the core profile repositories kept in the constructor for compatibility.
func WithIntegrationProfileReferenceRepo(uc ProfileManagementUseCase, repo domain.IntegrationProfileReferenceRepository) ProfileManagementUseCase {
	impl, ok := uc.(*profileManagementUseCase)
	if ok {
		impl.referenceRepo = repo
	}
	return uc
}

func NewProfileManagementUseCase(
	repo domain.IntegrationProfileRepository,
	demandRepo domain.DemandDocumentRepository,
	channelSyncRepo domain.ChannelSyncRepository,
	templateBindingRepo domain.ProfileTemplateBindingRepository,
	closureDecisionRepo domain.ChannelClosureDecisionRepository,
	executorProvider ExecutorProvider,
) ProfileManagementUseCase {
	return &profileManagementUseCase{
		repo:                repo,
		demandRepo:          demandRepo,
		channelSyncRepo:     channelSyncRepo,
		templateBindingRepo: templateBindingRepo,
		closureDecisionRepo: closureDecisionRepo,
		executorProvider:    executorProvider,
	}
}

// validSourceSurfaces is the shared value set for BusinessSurface / SourceSurface.
var validSourceSurfaces = map[string]bool{
	string(domain.SourceSurfaceMembership): true,
	string(domain.SourceSurfaceRetail):     true,
	string(domain.SourceSurfaceFactory):    true,
}

// validateProfileEnums checks SourceSurface and surface-specific constraints.
//
// SourceSurface is required and must be membership, retail, or factory.
// DemandKind is optional: empty is valid; a non-empty value must be a valid
// enum but is NOT paired 1:1 with SourceSurface (membership + retail_order is
// valid).
//
// membership/retail: existing 8 demand-strategy enum checks (non-empty → must be valid).
// factory: exempts the 8 demand-strategy checks; requires FactorySupplierPlatform and
// at least one factory capability flag.
func validateProfileEnums(input dto.CreateProfileInput) error {
	if input.SourceSurface == "" {
		return fmt.Errorf("source_surface is required")
	}
	if !validSourceSurfaces[input.SourceSurface] {
		return fmt.Errorf("invalid source_surface: %q", input.SourceSurface)
	}

	if input.SourceSurface == string(domain.SourceSurfaceFactory) {
		if strings.TrimSpace(input.FactorySupplierPlatform) == "" {
			return fmt.Errorf("factory surface requires a non-empty factory_supplier_platform")
		}
		if !input.SupportsExportSupplierOrder &&
			!input.SupportsImportProductCatalog &&
			!input.SupportsImportSupplierShipment {
			return fmt.Errorf("factory surface requires at least one factory capability (export_supplier_order, import_product_catalog, import_supplier_shipment)")
		}
		// Factory profiles do not use demand-side strategy enums.
		return nil
	}

	// membership / retail — demand-strategy enum validations (non-empty → must be valid).
	// DemandKind is not paired 1:1 with SourceSurface.
	validDemandKind := map[string]bool{
		"membership_entitlement": true,
		"retail_order":           true,
	}
	validInitialAllocationStrategy := map[string]bool{
		"policy_driven": true,
		"demand_driven": true,
	}
	validTrackingSyncMode := map[string]bool{
		"api_push":            true,
		"document_export":     true,
		"manual_confirmation": true,
		"unsupported":         true,
	}
	validClosurePolicy := map[string]bool{
		"close_after_sync":                true,
		"close_after_manual_confirmation": true,
		"close_after_shipment":            true,
	}
	validIdentityStrategy := map[string]bool{
		"platform_uid":                         true,
		"email":                                true,
		"external_buyer_id":                    true,
		IdentityStrategyOrderScopedProvisional: true,
	}
	validRecipientInputMode := map[string]bool{
		"none":              true,
		"platform_claim":    true,
		"external_form":     true,
		"manual_collection": true,
	}
	validReferenceStrategy := map[string]bool{
		"member_level":     true,
		"order_level":      true,
		"order_line_level": true,
	}
	validEntitlementAuthorityMode := map[string]bool{
		"local_policy":      true,
		"upstream_platform": true,
		"manual_grant_only": true,
	}

	// Empty DemandKind is valid; a leftover membership profile may still carry retail_order.
	if input.DemandKind != "" && !validDemandKind[input.DemandKind] {
		return fmt.Errorf("invalid demand_kind: %q", input.DemandKind)
	}
	if input.InitialAllocationStrategy != "" && !validInitialAllocationStrategy[input.InitialAllocationStrategy] {
		return fmt.Errorf("invalid initial_allocation_strategy: %q", input.InitialAllocationStrategy)
	}
	if input.TrackingSyncMode != "" && !validTrackingSyncMode[input.TrackingSyncMode] {
		return fmt.Errorf("invalid tracking_sync_mode: %q", input.TrackingSyncMode)
	}
	if input.ClosurePolicy != "" && !validClosurePolicy[input.ClosurePolicy] {
		return fmt.Errorf("invalid closure_policy: %q", input.ClosurePolicy)
	}
	if input.IdentityStrategy != "" && !validIdentityStrategy[input.IdentityStrategy] {
		return fmt.Errorf("invalid identity_strategy: %q", input.IdentityStrategy)
	}
	if input.RecipientInputMode != "" && !validRecipientInputMode[input.RecipientInputMode] {
		return fmt.Errorf("invalid recipient_input_mode: %q", input.RecipientInputMode)
	}
	if input.ReferenceStrategy != "" && !validReferenceStrategy[input.ReferenceStrategy] {
		return fmt.Errorf("invalid reference_strategy: %q", input.ReferenceStrategy)
	}
	if input.EntitlementAuthorityMode != "" && !validEntitlementAuthorityMode[input.EntitlementAuthorityMode] {
		return fmt.Errorf("invalid entitlement_authority_mode: %q", input.EntitlementAuthorityMode)
	}

	return nil
}

// validateExecutionReadiness checks that a profile's connector/mode configuration
// is sufficient for runtime execution ("write-means-executable" invariant).
//
// For executable modes, this should not stop at "non-empty connector_key".
// It should also ensure the current runtime registry can resolve that pair.
func validateExecutionReadiness(input dto.CreateProfileInput, executorProvider ExecutorProvider) error {
	switch input.TrackingSyncMode {
	case "manual_confirmation":
		if !input.AllowsManualClosure {
			return fmt.Errorf("tracking_sync_mode=manual_confirmation requires allows_manual_closure=true")
		}
	case "api_push", "document_export":
		if input.ConnectorKey == "" {
			return fmt.Errorf("tracking_sync_mode=%q requires a non-empty connector_key", input.TrackingSyncMode)
		}
		if executorProvider != nil {
			profile := &domain.IntegrationProfile{
				ProfileKey:       input.ProfileKey,
				TrackingSyncMode: input.TrackingSyncMode,
				ConnectorKey:     input.ConnectorKey,
			}
			if _, err := executorProvider.Resolve(profile); err != nil {
				return fmt.Errorf("execution readiness failed for tracking_sync_mode=%q and connector_key=%q: %w", input.TrackingSyncMode, input.ConnectorKey, err)
			}
		}
	}
	return nil
}

func (uc *profileManagementUseCase) CreateProfile(ctx context.Context, input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error) {
	if input.ProfileKey == "" {
		return nil, fmt.Errorf("profile_key is required")
	}

	// Friendly uniqueness pre-check before enum/readiness validation so the
	// operator gets a clear "already exists" rather than a raw DB unique error.
	if existing, err := uc.repo.FindByProfileKey(ctx, input.ProfileKey); err == nil && existing != nil {
		return nil, fmt.Errorf("profile_key %q already exists", input.ProfileKey)
	}

	if err := validateProfileEnums(input); err != nil {
		return nil, err
	}

	if err := validateExecutionReadiness(input, uc.executorProvider); err != nil {
		return nil, err
	}

	profile := profileFromCreateInput(input)

	if err := uc.repo.Create(ctx, profile); err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) UpdateProfile(ctx context.Context, input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error) {
	profile, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.ProfileKey) == "" {
		return nil, fmt.Errorf("profile_key must not be empty")
	}

	// When renaming, uniqueness must exclude the profile itself.
	if input.ProfileKey != "" && input.ProfileKey != profile.ProfileKey {
		if existing, findErr := uc.repo.FindByProfileKey(ctx, input.ProfileKey); findErr == nil && existing != nil && existing.ID != profile.ID {
			return nil, fmt.Errorf("profile_key %q already exists", input.ProfileKey)
		}
	}

	// Validate enums + readiness using a CreateProfileInput that carries the
	// full readiness-relevant field set (including ProfileKey / SourceChannel /
	// SourceSurface and factory capability flags).
	enumInput := dto.CreateProfileInput{
		ProfileKey:                     input.ProfileKey,
		SourceChannel:                  input.SourceChannel,
		SourceSurface:                  input.SourceSurface,
		DemandKind:                     input.DemandKind,
		InitialAllocationStrategy:      input.InitialAllocationStrategy,
		IdentityStrategy:               input.IdentityStrategy,
		EntitlementAuthorityMode:       input.EntitlementAuthorityMode,
		RecipientInputMode:             input.RecipientInputMode,
		ReferenceStrategy:              input.ReferenceStrategy,
		TrackingSyncMode:               input.TrackingSyncMode,
		ClosurePolicy:                  input.ClosurePolicy,
		SupportsPartialShipment:        input.SupportsPartialShipment,
		SupportsAPIImport:              input.SupportsAPIImport,
		SupportsAPIExport:              input.SupportsAPIExport,
		RequiresCarrierMapping:         input.RequiresCarrierMapping,
		RequiresExternalOrderNo:        input.RequiresExternalOrderNo,
		AllowsManualClosure:            input.AllowsManualClosure,
		SupportsExportSupplierOrder:    input.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   input.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: input.SupportsImportSupplierShipment,
		ConnectorKey:                   input.ConnectorKey,
		FactorySupplierPlatform:        input.FactorySupplierPlatform,
		SupportedLocales:               input.SupportedLocales,
		DefaultLocale:                  input.DefaultLocale,
		ExtraData:                      input.ExtraData,
	}
	if err := validateProfileEnums(enumInput); err != nil {
		return nil, err
	}
	if err := validateExecutionReadiness(enumInput, uc.executorProvider); err != nil {
		return nil, err
	}
	candidate := profileFromCreateInput(enumInput)
	candidate.ID = profile.ID
	bindings, err := uc.templateBindingRepo.ListByProfile(ctx, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("validate default template bindings: %w", err)
	}
	for _, binding := range bindings {
		if !binding.IsDefault {
			continue
		}
		if err := ValidateProfileDocumentType(candidate, binding.DocumentType); err != nil {
			return nil, fmt.Errorf("profile update would invalidate default binding %d for %q: %w", binding.ID, binding.DocumentType, err)
		}
	}

	profile.ProfileKey = input.ProfileKey
	profile.SourceChannel = input.SourceChannel
	profile.SourceSurface = input.SourceSurface
	profile.DemandKind = input.DemandKind
	profile.InitialAllocationStrategy = input.InitialAllocationStrategy
	profile.IdentityStrategy = input.IdentityStrategy
	profile.EntitlementAuthorityMode = input.EntitlementAuthorityMode
	profile.RecipientInputMode = input.RecipientInputMode
	profile.ReferenceStrategy = input.ReferenceStrategy
	profile.TrackingSyncMode = input.TrackingSyncMode
	profile.ClosurePolicy = input.ClosurePolicy
	profile.SupportsPartialShipment = input.SupportsPartialShipment
	profile.SupportsAPIImport = input.SupportsAPIImport
	profile.SupportsAPIExport = input.SupportsAPIExport
	profile.RequiresCarrierMapping = input.RequiresCarrierMapping
	profile.RequiresExternalOrderNo = input.RequiresExternalOrderNo
	profile.AllowsManualClosure = input.AllowsManualClosure
	profile.SupportsExportSupplierOrder = input.SupportsExportSupplierOrder
	profile.SupportsImportProductCatalog = input.SupportsImportProductCatalog
	profile.SupportsImportSupplierShipment = input.SupportsImportSupplierShipment
	profile.ConnectorKey = input.ConnectorKey
	profile.FactorySupplierPlatform = input.FactorySupplierPlatform
	profile.SupportedLocales = input.SupportedLocales
	profile.DefaultLocale = input.DefaultLocale
	profile.ExtraData = input.ExtraData

	if err := uc.repo.Update(ctx, profile); err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) DeleteProfile(ctx context.Context, id uint) error {
	count, err := uc.demandRepo.CountByIntegrationProfileID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check profile references: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete profile: %d demand documents still reference it", count)
	}

	syncCount, err := uc.channelSyncRepo.CountJobsByProfileID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check channel sync references: %w", err)
	}
	if syncCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by channel sync jobs")
	}

	bindingCount, err := uc.templateBindingRepo.CountByProfileID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check template binding references: %w", err)
	}
	if bindingCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by template bindings")
	}

	closureCount, err := uc.closureDecisionRepo.CountByProfileID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check closure decision references: %w", err)
	}
	if closureCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by closure decisions")
	}

	if uc.referenceRepo != nil {
		references, refErr := uc.referenceRepo.CountReferences(ctx, id)
		if refErr != nil {
			return fmt.Errorf("failed to check extended profile references: %w", refErr)
		}
		for _, kind := range []string{"carrier mappings", "supplier orders", "customer identities", "customer profile origins", "customer name observations"} {
			if references[kind] > 0 {
				return fmt.Errorf("cannot delete profile: %d %s still reference it", references[kind], kind)
			}
		}
	}

	return uc.repo.Delete(ctx, id)
}

func (uc *profileManagementUseCase) GetProfile(ctx context.Context, id uint) (*dto.IntegrationProfileDTO, error) {
	profile, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) ListProfiles(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
	profiles, err := uc.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.IntegrationProfileDTO, len(profiles))
	for i := range profiles {
		result[i] = profileToDTO(&profiles[i])
	}
	return result, nil
}

func (uc *profileManagementUseCase) SeedDefaultProfiles(ctx context.Context) ([]dto.IntegrationProfileDTO, error) {
	// One source-platform default. Empty DemandKind so both entitlement and
	// retail files can bind. Keep membership_default as the stable key;
	// retail_default is no longer seeded.
	defaults := []dto.CreateProfileInput{
		{
			ProfileKey:          "membership_default",
			SourceChannel:       "default",
			SourceSurface:       "membership",
			DemandKind:          "",
			TrackingSyncMode:    "document_export",
			ClosurePolicy:       "close_after_sync",
			AllowsManualClosure: false,
			ConnectorKey:        "eli.local_export",
		},
	}

	var result []dto.IntegrationProfileDTO
	for _, def := range defaults {
		_, err := uc.repo.FindByProfileKey(ctx, def.ProfileKey)
		if err == nil {
			continue
		}
		created, err := uc.CreateProfile(ctx, def)
		if err != nil {
			return nil, fmt.Errorf("create default profile %q: %w", def.ProfileKey, err)
		}
		result = append(result, *created)
	}
	return result, nil
}

// ---- helpers ----

// profileFromCreateInput maps a validated CreateProfileInput onto a domain entity.
// Shared by CreateProfile and seed paths that must not bypass field coverage.
func profileFromCreateInput(input dto.CreateProfileInput) *domain.IntegrationProfile {
	return &domain.IntegrationProfile{
		ProfileKey:                     input.ProfileKey,
		SourceChannel:                  input.SourceChannel,
		SourceSurface:                  input.SourceSurface,
		DemandKind:                     input.DemandKind,
		InitialAllocationStrategy:      input.InitialAllocationStrategy,
		IdentityStrategy:               input.IdentityStrategy,
		EntitlementAuthorityMode:       input.EntitlementAuthorityMode,
		RecipientInputMode:             input.RecipientInputMode,
		ReferenceStrategy:              input.ReferenceStrategy,
		TrackingSyncMode:               input.TrackingSyncMode,
		ClosurePolicy:                  input.ClosurePolicy,
		SupportsPartialShipment:        input.SupportsPartialShipment,
		SupportsAPIImport:              input.SupportsAPIImport,
		SupportsAPIExport:              input.SupportsAPIExport,
		RequiresCarrierMapping:         input.RequiresCarrierMapping,
		RequiresExternalOrderNo:        input.RequiresExternalOrderNo,
		AllowsManualClosure:            input.AllowsManualClosure,
		SupportsExportSupplierOrder:    input.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   input.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: input.SupportsImportSupplierShipment,
		ConnectorKey:                   input.ConnectorKey,
		FactorySupplierPlatform:        input.FactorySupplierPlatform,
		SupportedLocales:               input.SupportedLocales,
		DefaultLocale:                  input.DefaultLocale,
		ExtraData:                      input.ExtraData,
	}
}

func profileToDTO(p *domain.IntegrationProfile) dto.IntegrationProfileDTO {
	return dto.IntegrationProfileDTO{
		ID:                             p.ID,
		ProfileKey:                     p.ProfileKey,
		SourceChannel:                  p.SourceChannel,
		SourceSurface:                  p.SourceSurface,
		DemandKind:                     p.DemandKind,
		InitialAllocationStrategy:      p.InitialAllocationStrategy,
		IdentityStrategy:               p.IdentityStrategy,
		EntitlementAuthorityMode:       p.EntitlementAuthorityMode,
		RecipientInputMode:             p.RecipientInputMode,
		ReferenceStrategy:              p.ReferenceStrategy,
		TrackingSyncMode:               p.TrackingSyncMode,
		ClosurePolicy:                  p.ClosurePolicy,
		SupportsPartialShipment:        p.SupportsPartialShipment,
		SupportsAPIImport:              p.SupportsAPIImport,
		SupportsAPIExport:              p.SupportsAPIExport,
		RequiresCarrierMapping:         p.RequiresCarrierMapping,
		RequiresExternalOrderNo:        p.RequiresExternalOrderNo,
		AllowsManualClosure:            p.AllowsManualClosure,
		SupportsExportSupplierOrder:    p.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   p.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: p.SupportsImportSupplierShipment,
		ConnectorKey:                   p.ConnectorKey,
		FactorySupplierPlatform:        p.FactorySupplierPlatform,
		SupportedLocales:               p.SupportedLocales,
		DefaultLocale:                  p.DefaultLocale,
		ExtraData:                      p.ExtraData,
		CreatedAt:                      p.CreatedAt,
		UpdatedAt:                      p.UpdatedAt,
	}
}
