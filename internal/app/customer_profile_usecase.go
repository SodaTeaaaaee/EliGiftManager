package app

import (
	"fmt"
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
}

func NewCustomerProfileUseCase(
	profileRepo domain.CustomerProfileRepository,
	addressRepo domain.CustomerAddressRepository,
	settingsSvc *service.SettingsService,
	suggestionRepo domain.MergeSuggestionRepository,
) *CustomerProfileUseCase {
	return &CustomerProfileUseCase{
		profileRepo:    profileRepo,
		addressRepo:    addressRepo,
		settingsSvc:    settingsSvc,
		suggestionRepo: suggestionRepo,
	}
}

func (uc *CustomerProfileUseCase) ListCustomerProfiles(keyword, platform string, missingAddressOnly bool) ([]dto.CustomerProfileDTO, error) {
	profiles, err := uc.profileRepo.List()
	if err != nil {
		return nil, err
	}

	result := []dto.CustomerProfileDTO{}
	for _, p := range profiles {
		identities, err := uc.profileRepo.ListIdentitiesByProfile(p.ID)
		if err != nil {
			return nil, err
		}

		addresses, err := uc.addressRepo.ListByProfile(p.ID)
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
			if !matchKeyword {
				continue
			}
		}

		// Convert to DTO
		identDTOs := make([]dto.CustomerIdentityDTO, len(identities))
		for i, ident := range identities {
			identDTOs[i] = dto.CustomerIdentityDTO{
				ID:               ident.ID,
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
			ID:                 p.ID,
			DisplayName:        p.DisplayName,
			ProfileType:        p.ProfileType,
			ExtraData:          p.ExtraData,
			CreatedAt:          p.CreatedAt,
			UpdatedAt:          p.UpdatedAt,
			Identities:         identDTOs,
			Addresses:          addrDTOs,
			ActiveAddressCount: len(addresses),
		})
	}

	return result, nil
}

func (uc *CustomerProfileUseCase) GetCustomerProfile(id uint) (*dto.CustomerProfileDTO, error) {
	p, err := uc.profileRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	identities, err := uc.profileRepo.ListIdentitiesByProfile(p.ID)
	if err != nil {
		return nil, err
	}

	addresses, err := uc.addressRepo.ListByProfile(p.ID)
	if err != nil {
		return nil, err
	}

	identDTOs := make([]dto.CustomerIdentityDTO, len(identities))
	for i, ident := range identities {
		identDTOs[i] = dto.CustomerIdentityDTO{
			ID:               ident.ID,
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
		ID:                 p.ID,
		DisplayName:        p.DisplayName,
		ProfileType:        p.ProfileType,
		ExtraData:          p.ExtraData,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		Identities:         identDTOs,
		Addresses:          addrDTOs,
		ActiveAddressCount: len(addresses),
	}, nil
}

func (uc *CustomerProfileUseCase) CreateCustomerProfile(input dto.CreateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	now := time.Now().Format(time.RFC3339)
	p := &domain.CustomerProfile{
		DisplayName: input.DisplayName,
		ProfileType: input.ProfileType,
		ExtraData:   input.ExtraData,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.profileRepo.Create(p); err != nil {
		return nil, err
	}

	return uc.GetCustomerProfile(p.ID)
}

func (uc *CustomerProfileUseCase) UpdateCustomerProfile(input dto.UpdateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	p, err := uc.profileRepo.FindByID(input.ID)
	if err != nil {
		return nil, err
	}

	p.DisplayName = input.DisplayName
	p.ProfileType = input.ProfileType
	p.ExtraData = input.ExtraData
	p.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := uc.profileRepo.Update(p); err != nil {
		return nil, err
	}

	return uc.GetCustomerProfile(p.ID)
}

func (uc *CustomerProfileUseCase) DeleteCustomerProfile(id uint) error {
	return uc.profileRepo.SoftDelete(id)
}

func (uc *CustomerProfileUseCase) AddCustomerIdentity(input dto.CreateCustomerIdentityInput) (*dto.CustomerIdentityDTO, error) {
	now := time.Now().Format(time.RFC3339)
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

	if err := uc.profileRepo.CreateIdentity(ident); err != nil {
		return nil, err
	}

	// Trigger suggestion detection when identity is added
	_ = uc.DetectMergeSuggestions()

	return &dto.CustomerIdentityDTO{
		ID:               ident.ID,
		CustomerProfileID: ident.CustomerProfileID,
		IdentityPlatform:  ident.IdentityPlatform,
		IdentityValue:     ident.IdentityValue,
		IdentityType:      ident.IdentityType,
		IsPrimary:         ident.IsPrimary,
		ExtraData:         ident.ExtraData,
	}, nil
}

func (uc *CustomerProfileUseCase) DeleteCustomerIdentity(id uint) error {
	return uc.profileRepo.DeleteIdentity(id)
}

func (uc *CustomerProfileUseCase) GetMergeSuggestions() ([]dto.MergeSuggestionDTO, error) {
	// Detect fresh suggestions first
	_ = uc.DetectMergeSuggestions()

	suggestions, err := uc.suggestionRepo.ListPending()
	if err != nil {
		return nil, err
	}

	result := []dto.MergeSuggestionDTO{}
	for _, s := range suggestions {
		srcDTO, srcErr := uc.GetCustomerProfile(s.SourceProfileID)
		targetDTO, targetErr := uc.GetCustomerProfile(s.TargetProfileID)
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

func (uc *CustomerProfileUseCase) DismissMergeSuggestion(id uint) error {
	return uc.suggestionRepo.Dismiss(id)
}

func (uc *CustomerProfileUseCase) DetectMergeSuggestions() error {
	settings, err := uc.settingsSvc.Load()
	if err != nil {
		return err
	}
	if !settings.AutoMergeCrossPlatform {
		return nil
	}

	// 1. If ByEmail is enabled, find duplicates by email
	if settings.AutoMergeByEmail {
		emailGroups, err := uc.suggestionRepo.FindEmailDuplicates()
		if err == nil {
			for _, eg := range emailGroups {
				uc.createSuggestionsFromIDs(eg.ProfileIDs, fmt.Sprintf("相同邮箱: %s", eg.Key))
			}
		}
	}

	// 2. If ByPhone is enabled, find duplicates by phone
	if settings.AutoMergeByPhone {
		phoneGroups, err := uc.suggestionRepo.FindPhoneDuplicates()
		if err == nil {
			for _, pg := range phoneGroups {
				uc.createSuggestionsFromIDs(pg.ProfileIDs, fmt.Sprintf("相同电话: %s", pg.Key))
			}
		}
	}

	return nil
}

func (uc *CustomerProfileUseCase) createSuggestionsFromIDs(idsStr, reason string) {
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
		existing, _ := uc.suggestionRepo.CountBySourceAndTarget(sourceID, targetID)
		if existing == 0 {
			suggestion := domain.MergeSuggestion{
				SourceProfileID: sourceID,
				TargetProfileID: targetID,
				Reason:          reason,
				Status:          "pending",
			}
			_ = uc.suggestionRepo.Create(&suggestion)
		}
	}
}

func (uc *CustomerProfileUseCase) SaveSettings(settings dto.SystemSettingsDTO) error {
	return uc.settingsSvc.Save(&service.SystemSettings{
		AutoMergeCrossPlatform: settings.AutoMergeCrossPlatform,
		AutoMergeByEmail:       settings.AutoMergeByEmail,
		AutoMergeByPhone:       settings.AutoMergeByPhone,
	})
}

func (uc *CustomerProfileUseCase) GetSettings() (dto.SystemSettingsDTO, error) {
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
