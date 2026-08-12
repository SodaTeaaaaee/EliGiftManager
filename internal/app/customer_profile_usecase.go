package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

type CustomerProfileUseCase struct {
	profileRepo    domain.CustomerProfileRepository
	addressRepo    domain.CustomerAddressRepository
	settingsSvc    *service.SettingsService
	suggestionRepo domain.MergeSuggestionRepository
	nameService    *CustomerNameObservationService
	nativeProfiles domain.CustomerProfileNativeRepository
	originRepo     domain.CustomerProfileOriginRepository
	originReads    domain.CustomerProfileOriginReadRepository
}

func WithCustomerProfileOrigins(uc *CustomerProfileUseCase, repo domain.CustomerProfileOriginRepository) *CustomerProfileUseCase {
	uc.originRepo = repo
	uc.originReads, _ = repo.(domain.CustomerProfileOriginReadRepository)
	return uc
}

func NewCustomerProfileUseCase(
	profileRepo domain.CustomerProfileRepository,
	addressRepo domain.CustomerAddressRepository,
	settingsSvc *service.SettingsService,
	suggestionRepo domain.MergeSuggestionRepository,
	nameServices ...*CustomerNameObservationService,
) *CustomerProfileUseCase {
	uc := &CustomerProfileUseCase{
		profileRepo:    profileRepo,
		addressRepo:    addressRepo,
		settingsSvc:    settingsSvc,
		suggestionRepo: suggestionRepo,
	}
	uc.nativeProfiles, _ = profileRepo.(domain.CustomerProfileNativeRepository)
	if len(nameServices) > 0 {
		uc.nameService = nameServices[0]
	}
	return uc
}

func (uc *CustomerProfileUseCase) ListCustomerProfiles(ctx context.Context, keyword, platform string, missingAddressOnly bool) ([]dto.CustomerProfileDTO, error) {
	profiles, err := uc.profileRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := []dto.CustomerProfileDTO{}
	for _, p := range profiles {
		identities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, p.ID)
		if err != nil {
			return nil, err
		}

		addresses, err := uc.addressRepo.ListByProfile(ctx, p.ID)
		if err != nil {
			return nil, err
		}

		// Filter by platform
		if platform != "" {
			hasPlatform := false
			for _, ident := range identities {
				if strings.EqualFold(ident.IdentityPlatform, platform) {
					hasPlatform = true
					break
				}
			}
			if !hasPlatform {
				continue
			}
		}

		// Filter by missing address only
		if missingAddressOnly && len(addresses) > 0 {
			continue
		}

		matchedHistoricalName := ""
		// Filter by keyword
		if keyword != "" {
			matchKeyword := false
			if strings.Contains(strings.ToLower(p.DisplayName), strings.ToLower(keyword)) {
				matchKeyword = true
			} else {
				for _, ident := range identities {
					if strings.Contains(strings.ToLower(ident.IdentityValue), strings.ToLower(keyword)) {
						matchKeyword = true
						break
					}
				}
			}
			if !matchKeyword && uc.nameService != nil {
				observations, err := uc.nameService.ListObservations(ctx, p.ID)
				if err != nil {
					return nil, err
				}
				matchedHistoricalName = historicalNameMatch(observations, keyword)
				matchKeyword = matchedHistoricalName != ""
			}
			if !matchKeyword {
				continue
			}
		}

		// Convert to DTO
		identDTOs := make([]dto.CustomerIdentityDTO, len(identities))
		for i, ident := range identities {
			identDTOs[i] = dto.CustomerIdentityDTO{
				ID:                ident.ID,
				CustomerProfileID: ident.CustomerProfileID,
				IdentityPlatform:  ident.IdentityPlatform,
				IdentityValue:     ident.IdentityValue,
				IdentityType:      ident.IdentityType,
				IsPrimary:         ident.IsPrimary,
				ExtraData:         ident.ExtraData,
			}
		}

		addrDTOs := make([]dto.CustomerAddressDTO, len(addresses))
		for i, addr := range addresses {
			addrDTOs[i] = dto.CustomerAddressDTO{
				ID:                addr.ID,
				CustomerProfileID: addr.CustomerProfileID,
				Label:             addr.Label,
				RecipientName:     addr.RecipientName,
				Phone:             addr.Phone,
				Country:           addr.Country,
				Province:          addr.Province,
				City:              addr.City,
				District:          addr.District,
				AddressLine1:      addr.AddressLine1,
				AddressLine2:      addr.AddressLine2,
				PostalCode:        addr.PostalCode,
				IsDefault:         addr.IsDefault,
				IsTest:            addr.IsTest,
				ValidationStatus:  addr.ValidationStatus,
				ValidationDetail:  addr.ValidationDetail,
				ExtraData:         addr.ExtraData,
				CreatedAt:         addr.CreatedAt,
				UpdatedAt:         addr.UpdatedAt,
			}
		}

		result = append(result, dto.CustomerProfileDTO{
			ID:                       p.ID,
			DisplayName:              p.DisplayName,
			ProfileType:              p.ProfileType,
			Status:                   p.Status,
			MergedIntoProfileID:      p.MergedIntoProfileID,
			RowVersion:               p.RowVersion,
			DisplayNameMode:          p.DisplayNameMode,
			DisplayNameObservationID: p.DisplayNameObservationID,
			MatchedHistoricalName:    matchedHistoricalName,
			ExtraData:                p.ExtraData,
			CreatedAt:                p.CreatedAt,
			UpdatedAt:                p.UpdatedAt,
			Identities:               identDTOs,
			Addresses:                addrDTOs,
			ActiveAddressCount:       len(addresses),
		})
	}

	return result, nil
}

func (uc *CustomerProfileUseCase) GetCustomerProfile(ctx context.Context, id uint) (*dto.CustomerProfileDTO, error) {
	p, err := uc.findProfileForRead(ctx, id)
	if err != nil {
		return nil, err
	}

	identities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	addresses, err := uc.addressRepo.ListByProfile(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	identDTOs := make([]dto.CustomerIdentityDTO, len(identities))
	for i, ident := range identities {
		identDTOs[i] = dto.CustomerIdentityDTO{
			ID:                ident.ID,
			CustomerProfileID: ident.CustomerProfileID,
			IdentityPlatform:  ident.IdentityPlatform,
			IdentityValue:     ident.IdentityValue,
			IdentityType:      ident.IdentityType,
			IsPrimary:         ident.IsPrimary,
			ExtraData:         ident.ExtraData,
		}
	}

	addrDTOs := make([]dto.CustomerAddressDTO, len(addresses))
	for i, addr := range addresses {
		addrDTOs[i] = dto.CustomerAddressDTO{
			ID:                addr.ID,
			CustomerProfileID: addr.CustomerProfileID,
			Label:             addr.Label,
			RecipientName:     addr.RecipientName,
			Phone:             addr.Phone,
			Country:           addr.Country,
			Province:          addr.Province,
			City:              addr.City,
			District:          addr.District,
			AddressLine1:      addr.AddressLine1,
			AddressLine2:      addr.AddressLine2,
			PostalCode:        addr.PostalCode,
			IsDefault:         addr.IsDefault,
			IsTest:            addr.IsTest,
			ValidationStatus:  addr.ValidationStatus,
			ValidationDetail:  addr.ValidationDetail,
			ExtraData:         addr.ExtraData,
			CreatedAt:         addr.CreatedAt,
			UpdatedAt:         addr.UpdatedAt,
		}
	}

	return &dto.CustomerProfileDTO{
		ID:                       p.ID,
		DisplayName:              p.DisplayName,
		ProfileType:              p.ProfileType,
		Status:                   p.Status,
		MergedIntoProfileID:      p.MergedIntoProfileID,
		RowVersion:               p.RowVersion,
		DisplayNameMode:          p.DisplayNameMode,
		DisplayNameObservationID: p.DisplayNameObservationID,
		ExtraData:                p.ExtraData,
		CreatedAt:                p.CreatedAt,
		UpdatedAt:                p.UpdatedAt,
		Identities:               identDTOs,
		Addresses:                addrDTOs,
		ActiveAddressCount:       len(addresses),
	}, nil
}

func (uc *CustomerProfileUseCase) CreateCustomerProfile(ctx context.Context, input dto.CreateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	now := time.Now()
	p := &domain.CustomerProfile{
		DisplayName:     input.DisplayName,
		ProfileType:     input.ProfileType,
		Status:          domain.CustomerProfileStatusActive,
		RowVersion:      1,
		DisplayNameMode: domain.DisplayNameModeAuto,
		ExtraData:       input.ExtraData,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.profileRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	return uc.GetCustomerProfile(ctx, p.ID)
}

func (uc *CustomerProfileUseCase) UpdateCustomerProfile(ctx context.Context, input dto.UpdateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	p, err := uc.findProfileForRead(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if p.Status == domain.CustomerProfileStatusMerged || p.MergedIntoProfileID != nil {
		return nil, fmt.Errorf("update customer profile: %w: profile=%d target=%v", ErrCustomerProfileMerged, p.ID, p.MergedIntoProfileID)
	}
	expected := input.ExpectedRowVersion
	if expected == 0 {
		expected = p.RowVersion
	}
	if p.RowVersion != expected {
		return nil, fmt.Errorf("update customer profile: %w: profile=%d expected=%d actual=%d", ErrCustomerProfileStale, p.ID, expected, p.RowVersion)
	}
	if input.DisplayName != p.DisplayName {
		if uc.nameService == nil {
			return nil, errors.New("update customer profile: customer name history support is unavailable")
		}
		actor := strings.TrimSpace(input.ActorRef)
		if actor == "" {
			actor = "local_user"
		}
		key := strings.TrimSpace(input.IdempotencyKey)
		if key == "" {
			key = manualNameUpdateKey(p.ID, expected, input.DisplayName)
		}
		if err := uc.nameService.PinExpected(ctx, p.ID, input.DisplayName, expected, actor, key, time.Now().UTC()); err != nil {
			return nil, err
		}
		expected++
	}
	if input.ProfileType != p.ProfileType || input.ExtraData != p.ExtraData {
		if uc.nativeProfiles == nil {
			return nil, errors.New("update customer profile: profile metadata CAS is unavailable")
		}
		updated, err := uc.nativeProfiles.UpdateProfileMetadataCAS(ctx, p.ID, expected, input.ProfileType, input.ExtraData)
		if err != nil {
			return nil, err
		}
		if !updated {
			return nil, fmt.Errorf("update customer profile: %w: profile=%d expected=%d", ErrCustomerProfileStale, p.ID, expected)
		}
	}

	return uc.GetCustomerProfile(ctx, p.ID)
}

func (uc *CustomerProfileUseCase) DeleteCustomerProfile(ctx context.Context, id uint) error {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return err
	}
	if profile, err := uc.findProfileForRead(ctx, id); err == nil && (profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil) {
		return fmt.Errorf("delete customer profile: %w: profile=%d target=%v", ErrCustomerProfileMerged, profile.ID, profile.MergedIntoProfileID)
	}
	return uc.profileRepo.SoftDelete(ctx, id)
}

func (uc *CustomerProfileUseCase) AddCustomerIdentity(ctx context.Context, input dto.CreateCustomerIdentityInput) (*dto.CustomerIdentityDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	if profile, err := uc.findProfileForRead(ctx, input.CustomerProfileID); err != nil {
		return nil, err
	} else if profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil {
		return nil, fmt.Errorf("add customer identity: %w: profile=%d target=%v", ErrCustomerProfileMerged, profile.ID, profile.MergedIntoProfileID)
	}
	now := time.Now()
	ident := &domain.CustomerIdentity{
		CustomerProfileID: input.CustomerProfileID,
		IdentityPlatform:  input.IdentityPlatform,
		IdentityValue:     input.IdentityValue,
		IdentityType:      input.IdentityType,
		IsPrimary:         input.IsPrimary,
		ExtraData:         input.ExtraData,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := uc.profileRepo.CreateIdentity(ctx, ident); err != nil {
		return nil, err
	}

	return &dto.CustomerIdentityDTO{
		ID:                ident.ID,
		CustomerProfileID: ident.CustomerProfileID,
		IdentityPlatform:  ident.IdentityPlatform,
		IdentityValue:     ident.IdentityValue,
		IdentityType:      ident.IdentityType,
		IsPrimary:         ident.IsPrimary,
		ExtraData:         ident.ExtraData,
	}, nil
}

func (uc *CustomerProfileUseCase) ListCustomerNameObservations(ctx context.Context, profileID uint) ([]dto.CustomerNameObservationDTO, error) {
	profile, err := uc.findProfileForRead(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if uc.nameService == nil {
		return nil, errors.New("customer name history support is unavailable")
	}
	observations, err := uc.nameService.ListObservations(ctx, profileID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CustomerNameObservationDTO, 0, len(observations))
	for i := range observations {
		if !observations[i].IsActive {
			continue
		}
		originProfileID := observations[i].OriginProfileID
		if originProfileID == 0 {
			originProfileID = observations[i].CustomerProfileID
		}
		result = append(result, dto.CustomerNameObservationDTO{
			ID: observations[i].ID, Kind: observations[i].NameKind, Value: observations[i].Name,
			Source: observations[i].Authority, FirstSeenAt: observations[i].FirstSeenAt,
			LastSeenAt: observations[i].LastSeenAt, Count: observations[i].ObservationCount,
			IsDisplayNameSource: profile.DisplayNameObservationID != nil && *profile.DisplayNameObservationID == observations[i].ID,
			OriginProfileID:     originProfileID,
		})
	}
	return result, nil
}

func (uc *CustomerProfileUseCase) ListCustomerProfileOrigins(ctx context.Context, profileID uint) ([]dto.CustomerProfileOriginDTO, error) {
	profile, err := uc.findProfileForRead(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if uc.originRepo == nil {
		return nil, errors.New("customer profile origin history support is unavailable")
	}
	var origins []domain.CustomerProfileOrigin
	if (profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil) && uc.originReads != nil {
		origins, err = uc.originReads.ListForProfileRead(ctx, profileID)
	} else {
		origins, err = uc.originRepo.ListByProfile(ctx, profileID)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(origins, func(i, j int) bool { return origins[i].ID < origins[j].ID })
	result := make([]dto.CustomerProfileOriginDTO, len(origins))
	for i := range origins {
		result[i] = dto.CustomerProfileOriginDTO{ID: origins[i].ID, CustomerProfileID: origins[i].CustomerProfileID,
			OriginKind: origins[i].OriginKind, SourceIntegrationProfileID: origins[i].SourceIntegrationProfileID,
			ExternalRef: origins[i].ExternalRef, SourceDocumentID: origins[i].SourceDocumentID,
			LastSeenAt: origins[i].LastSeenAt, CreatedAt: origins[i].CreatedAt}
	}
	return result, nil
}

func (uc *CustomerProfileUseCase) PinCustomerDisplayName(ctx context.Context, input dto.PinCustomerDisplayNameInput) (*dto.CustomerProfileDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	profile, err := uc.findProfileForRead(ctx, input.ProfileID)
	if err != nil {
		return nil, err
	}
	if profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil {
		return nil, fmt.Errorf("pin customer display name: %w: profile=%d target=%v", ErrCustomerProfileMerged, profile.ID, profile.MergedIntoProfileID)
	}
	if uc.nameService == nil {
		return nil, errors.New("customer name history support is unavailable")
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return nil, errors.New("pin customer display name: idempotencyKey is required")
	}
	actor := strings.TrimSpace(input.ActorRef)
	if actor == "" {
		actor = "local_user"
	}
	if err := uc.nameService.PinExpected(ctx, input.ProfileID, input.Name, input.ExpectedRowVersion, actor, input.IdempotencyKey, time.Now().UTC()); err != nil {
		return nil, err
	}
	return uc.GetCustomerProfile(ctx, input.ProfileID)
}

func (uc *CustomerProfileUseCase) UnpinCustomerDisplayName(ctx context.Context, input dto.UnpinCustomerDisplayNameInput) (*dto.CustomerProfileDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	profile, err := uc.findProfileForRead(ctx, input.ProfileID)
	if err != nil {
		return nil, err
	}
	if profile.Status == domain.CustomerProfileStatusMerged || profile.MergedIntoProfileID != nil {
		return nil, fmt.Errorf("unpin customer display name: %w: profile=%d target=%v", ErrCustomerProfileMerged, profile.ID, profile.MergedIntoProfileID)
	}
	if uc.nameService == nil {
		return nil, errors.New("customer name history support is unavailable")
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return nil, errors.New("unpin customer display name: idempotencyKey is required")
	}
	actor := strings.TrimSpace(input.ActorRef)
	if actor == "" {
		actor = "local_user"
	}
	if err := uc.nameService.UnpinExpected(ctx, input.ProfileID, input.ExpectedRowVersion, actor, input.IdempotencyKey, time.Now().UTC()); err != nil {
		return nil, err
	}
	return uc.GetCustomerProfile(ctx, input.ProfileID)
}

func (uc *CustomerProfileUseCase) findProfileForRead(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	profile, err := uc.profileRepo.FindByID(ctx, id)
	if err == nil || uc.nativeProfiles == nil {
		return profile, err
	}
	includingDeleted, includingDeletedErr := uc.nativeProfiles.FindByIDIncludingDeleted(ctx, id)
	if includingDeletedErr != nil {
		return nil, err
	}
	if includingDeleted.Status == domain.CustomerProfileStatusMerged || includingDeleted.MergedIntoProfileID != nil {
		return includingDeleted, nil
	}
	return nil, err
}

func historicalNameMatch(observations []domain.CustomerNameObservation, keyword string) string {
	normalized := normalizeCustomerName(keyword)
	raw := strings.ToLower(strings.TrimSpace(keyword))
	for i := len(observations) - 1; i >= 0; i-- {
		if !observations[i].IsActive {
			continue
		}
		if (normalized != "" && strings.Contains(observations[i].NormalizedName, normalized)) ||
			(raw != "" && strings.Contains(strings.ToLower(observations[i].Name), raw)) {
			return observations[i].Name
		}
	}
	return ""
}

func manualNameUpdateKey(profileID uint, expected uint64, name string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return fmt.Sprintf("manual-profile-update:%d:%d:%s", profileID, expected, hex.EncodeToString(sum[:8]))
}

func (uc *CustomerProfileUseCase) DeleteCustomerIdentity(ctx context.Context, id uint) error {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return err
	}
	return uc.profileRepo.DeleteIdentity(ctx, id)
}

func (uc *CustomerProfileUseCase) GetMergeSuggestions(ctx context.Context) ([]dto.MergeSuggestionDTO, error) {
	suggestions, err := uc.suggestionRepo.ListPending(ctx)
	if err != nil {
		return nil, err
	}

	result := []dto.MergeSuggestionDTO{}
	for _, s := range suggestions {
		srcDTO, srcErr := uc.GetCustomerProfile(ctx, s.SourceProfileID)
		targetDTO, targetErr := uc.GetCustomerProfile(ctx, s.TargetProfileID)
		if srcErr != nil || targetErr != nil {
			continue // skip if either profile was deleted
		}

		result = append(result, dto.MergeSuggestionDTO{
			ID:              s.ID,
			SourceProfileID: s.SourceProfileID,
			TargetProfileID: s.TargetProfileID,
			Reason:          s.Reason,
			Status:          s.Status,
			SourceProfile:   *srcDTO,
			TargetProfile:   *targetDTO,
		})
	}

	return result, nil
}

func (uc *CustomerProfileUseCase) DismissMergeSuggestion(ctx context.Context, id uint) error {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureCandidateScan); err != nil {
		return err
	}
	return uc.suggestionRepo.Dismiss(ctx, id)
}

func (uc *CustomerProfileUseCase) DetectMergeSuggestions(ctx context.Context) error {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureCandidateScan); err != nil {
		return err
	}
	settings, err := uc.settingsSvc.Load()
	if err != nil {
		return err
	}
	if !settings.AutoMergeCrossPlatform {
		return nil
	}

	// 1. If ByEmail is enabled, find duplicates by email
	if settings.AutoMergeByEmail {
		emailGroups, err := uc.suggestionRepo.FindEmailDuplicates(ctx)
		if err == nil {
			for _, eg := range emailGroups {
				uc.createSuggestionsFromIDs(ctx, eg.ProfileIDs, fmt.Sprintf("相同邮箱: %s", eg.Key))
			}
		}
	}

	// 2. If ByPhone is enabled, find duplicates by phone
	if settings.AutoMergeByPhone {
		phoneGroups, err := uc.suggestionRepo.FindPhoneDuplicates(ctx)
		if err == nil {
			for _, pg := range phoneGroups {
				uc.createSuggestionsFromIDs(ctx, pg.ProfileIDs, fmt.Sprintf("相同电话: %s", pg.Key))
			}
		}
	}

	return nil
}

func (uc *CustomerProfileUseCase) createSuggestionsFromIDs(ctx context.Context, idsStr, reason string) {
	parts := strings.Split(idsStr, ",")
	var profileIDs []uint
	for _, p := range parts {
		var id uint
		if _, err := fmt.Sscanf(p, "%d", &id); err == nil {
			profileIDs = append(profileIDs, id)
		}
	}
	if len(profileIDs) < 2 {
		return
	}

	// Target profile is the smallest ID (earliest created)
	targetID := profileIDs[0]
	for _, id := range profileIDs {
		if id < targetID {
			targetID = id
		}
	}

	for _, sourceID := range profileIDs {
		if sourceID == targetID {
			continue
		}

		// Check if suggestion already exists
		existing, _ := uc.suggestionRepo.CountBySourceAndTarget(ctx, sourceID, targetID)
		if existing == 0 {
			suggestion := domain.MergeSuggestion{
				SourceProfileID: sourceID,
				TargetProfileID: targetID,
				Reason:          reason,
				Status:          "pending",
			}
			_ = uc.suggestionRepo.Create(ctx, &suggestion)
		}
	}
}

func (uc *CustomerProfileUseCase) SaveSettings(ctx context.Context, settings dto.SystemSettingsDTO) error {
	return uc.settingsSvc.Save(&service.SystemSettings{
		AutoMergeCrossPlatform: settings.AutoMergeCrossPlatform,
		AutoMergeByEmail:       settings.AutoMergeByEmail,
		AutoMergeByPhone:       settings.AutoMergeByPhone,
	})
}

func (uc *CustomerProfileUseCase) GetSettings(ctx context.Context) (dto.SystemSettingsDTO, error) {
	s, err := uc.settingsSvc.Load()
	if err != nil {
		return dto.SystemSettingsDTO{}, err
	}
	return dto.SystemSettingsDTO{
		AutoMergeCrossPlatform: s.AutoMergeCrossPlatform,
		AutoMergeByEmail:       s.AutoMergeByEmail,
		AutoMergeByPhone:       s.AutoMergeByPhone,
	}, nil
}
