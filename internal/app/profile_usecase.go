package app

import (
	"fmt"

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

// validateProfileEnums checks that all non-empty strategy enum fields contain valid values.
func validateProfileEnums(input dto.CreateProfileInput) error {
	validDemandKind := map[string]bool{
		"membership_entitlement": true,
		"retail_order":           true,
	}
	validInitialAllocationStrategy := map[string]bool{
		"policy_driven": true,
		"demand_driven": true,
	}
	validTrackingSyncMode := map[string]bool{
		"api_push":              true,
		"document_export":       true,
		"manual_confirmation":   true,
		"unsupported":           true,
	}
	validClosurePolicy := map[string]bool{
		"close_after_sync":                  true,
		"close_after_manual_confirmation":   true,
		"close_after_shipment":              true,
	}
	validIdentityStrategy := map[string]bool{
		"platform_uid":      true,
		"email":             true,
		"external_buyer_id": true,
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

func (uc *profileManagementUseCase) CreateProfile(input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error) {
	if input.ProfileKey == "" {
		return nil, fmt.Errorf("profile_key is required")
	}

	if err := validateProfileEnums(input); err != nil {
		return nil, err
	}

	if err := validateExecutionReadiness(input, uc.executorProvider); err != nil {
		return nil, err
	}

	profile := &domain.IntegrationProfile{
		ProfileKey:                input.ProfileKey,
		SourceChannel:             input.SourceChannel,
		SourceSurface:             input.SourceSurface,
		DemandKind:                input.DemandKind,
		InitialAllocationStrategy: input.InitialAllocationStrategy,
		IdentityStrategy:          input.IdentityStrategy,
		EntitlementAuthorityMode:  input.EntitlementAuthorityMode,
		RecipientInputMode:        input.RecipientInputMode,
		ReferenceStrategy:         input.ReferenceStrategy,
		TrackingSyncMode:          input.TrackingSyncMode,
		ClosurePolicy:             input.ClosurePolicy,
		SupportsPartialShipment:   input.SupportsPartialShipment,
		SupportsAPIImport:         input.SupportsAPIImport,
		SupportsAPIExport:         input.SupportsAPIExport,
		RequiresCarrierMapping:    input.RequiresCarrierMapping,
		RequiresExternalOrderNo:   input.RequiresExternalOrderNo,
		AllowsManualClosure:       input.AllowsManualClosure,
		ConnectorKey:              input.ConnectorKey,
		SupportedLocales:          input.SupportedLocales,
		DefaultLocale:             input.DefaultLocale,
		ExtraData:                 input.ExtraData,
	}

	if err := uc.repo.Create(profile); err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) UpdateProfile(input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error) {
	profile, err := uc.repo.FindByID(input.ID)
	if err != nil {
		return nil, err
	}

	// Validate enums using a CreateProfileInput (same field set)
	enumInput := dto.CreateProfileInput{
		DemandKind:                input.DemandKind,
		InitialAllocationStrategy: input.InitialAllocationStrategy,
		IdentityStrategy:          input.IdentityStrategy,
		EntitlementAuthorityMode:  input.EntitlementAuthorityMode,
		RecipientInputMode:        input.RecipientInputMode,
		ReferenceStrategy:         input.ReferenceStrategy,
		TrackingSyncMode:          input.TrackingSyncMode,
		ClosurePolicy:             input.ClosurePolicy,
		AllowsManualClosure:       input.AllowsManualClosure,
		ConnectorKey:              input.ConnectorKey,
	}
	if err := validateProfileEnums(enumInput); err != nil {
		return nil, err
	}
	if err := validateExecutionReadiness(enumInput, uc.executorProvider); err != nil {
		return nil, err
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
	profile.ConnectorKey = input.ConnectorKey
	profile.SupportedLocales = input.SupportedLocales
	profile.DefaultLocale = input.DefaultLocale
	profile.ExtraData = input.ExtraData

	if err := uc.repo.Update(profile); err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) DeleteProfile(id uint) error {
	count, err := uc.demandRepo.CountByProfileID(id)
	if err != nil {
		return fmt.Errorf("failed to check profile references: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete profile: %d demand documents still reference it", count)
	}

	syncCount, err := uc.channelSyncRepo.CountJobsByProfileID(id)
	if err != nil {
		return fmt.Errorf("failed to check channel sync references: %w", err)
	}
	if syncCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by channel sync jobs")
	}

	bindingCount, err := uc.templateBindingRepo.CountByProfileID(id)
	if err != nil {
		return fmt.Errorf("failed to check template binding references: %w", err)
	}
	if bindingCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by template bindings")
	}

	closureCount, err := uc.closureDecisionRepo.CountByProfileID(id)
	if err != nil {
		return fmt.Errorf("failed to check closure decision references: %w", err)
	}
	if closureCount > 0 {
		return fmt.Errorf("cannot delete profile: referenced by closure decisions")
	}

	return uc.repo.Delete(id)
}

func (uc *profileManagementUseCase) GetProfile(id uint) (*dto.IntegrationProfileDTO, error) {
	profile, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	d := profileToDTO(profile)
	return &d, nil
}

func (uc *profileManagementUseCase) ListProfiles() ([]dto.IntegrationProfileDTO, error) {
	profiles, err := uc.repo.List()
	if err != nil {
		return nil, err
	}
	result := make([]dto.IntegrationProfileDTO, len(profiles))
	for i := range profiles {
		result[i] = profileToDTO(&profiles[i])
	}
	return result, nil
}

func (uc *profileManagementUseCase) SeedDefaultProfiles() ([]dto.IntegrationProfileDTO, error) {
	defaults := []dto.CreateProfileInput{
		{
			ProfileKey:                "membership_default",
			SourceChannel:             "default_membership_channel",
			SourceSurface:             "membership",
			DemandKind:                "membership_entitlement",
			InitialAllocationStrategy: "policy_driven",
			IdentityStrategy:          "platform_uid",
			EntitlementAuthorityMode:  "local_policy",
			RecipientInputMode:        "none",
			ReferenceStrategy:         "member_level",
			TrackingSyncMode:          "manual_confirmation",
			ClosurePolicy:             "close_after_manual_confirmation",
			AllowsManualClosure:       true,
		},
		{
			ProfileKey:                "retail_default",
			SourceChannel:             "default_retail_channel",
			SourceSurface:             "retail",
			DemandKind:                "retail_order",
			InitialAllocationStrategy: "demand_driven",
			IdentityStrategy:          "email",
			EntitlementAuthorityMode:  "local_policy",
			RecipientInputMode:        "none",
			ReferenceStrategy:         "order_line_level",
			TrackingSyncMode:          "document_export",
			ClosurePolicy:             "close_after_sync",
			AllowsManualClosure:       false,
			ConnectorKey:              "eli.local_export",
		},
	}

	var result []dto.IntegrationProfileDTO
	for _, def := range defaults {
		_, err := uc.repo.FindByProfileKey(def.ProfileKey)
		if err == nil {
			continue
		}
		created, err := uc.CreateProfile(def)
		if err != nil {
			return nil, fmt.Errorf("create default profile %q: %w", def.ProfileKey, err)
		}
		result = append(result, *created)
	}
	return result, nil
}

// ---- helpers ----

func profileToDTO(p *domain.IntegrationProfile) dto.IntegrationProfileDTO {
	return dto.IntegrationProfileDTO{
		ID:                        p.ID,
		ProfileKey:                p.ProfileKey,
		SourceChannel:             p.SourceChannel,
		SourceSurface:             p.SourceSurface,
		DemandKind:                p.DemandKind,
		InitialAllocationStrategy: p.InitialAllocationStrategy,
		IdentityStrategy:          p.IdentityStrategy,
		EntitlementAuthorityMode:  p.EntitlementAuthorityMode,
		RecipientInputMode:        p.RecipientInputMode,
		ReferenceStrategy:         p.ReferenceStrategy,
		TrackingSyncMode:          p.TrackingSyncMode,
		ClosurePolicy:             p.ClosurePolicy,
		SupportsPartialShipment:   p.SupportsPartialShipment,
		SupportsAPIImport:         p.SupportsAPIImport,
		SupportsAPIExport:         p.SupportsAPIExport,
		RequiresCarrierMapping:    p.RequiresCarrierMapping,
		RequiresExternalOrderNo:   p.RequiresExternalOrderNo,
		AllowsManualClosure:       p.AllowsManualClosure,
		ConnectorKey:              p.ConnectorKey,
		SupportedLocales:          p.SupportedLocales,
		DefaultLocale:             p.DefaultLocale,
		ExtraData:                 p.ExtraData,
		CreatedAt:                 p.CreatedAt,
		UpdatedAt:                 p.UpdatedAt,
	}
}
