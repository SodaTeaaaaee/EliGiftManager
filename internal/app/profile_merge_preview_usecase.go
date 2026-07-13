package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ProfileMergePreviewUseCase computes a read-only conflict-detail preview for
// a prospective MergeProfiles call: both sides' identities/addresses plus a
// highlighted conflict list, so the operator can review before committing
// (plan 5.2). It is a sibling to ProfileMergeUseCase — declared as its own
// interface in its own file rather than added to profile_merge_usecase.go —
// and never mutates anything.
type ProfileMergePreviewUseCase interface {
	PreviewMergeProfiles(ctx context.Context, sourceProfileID, targetProfileID uint) (*dto.MergeProfilesPreviewResult, error)
}

type profileMergePreviewUseCase struct {
	profileRepo domain.CustomerProfileRepository
	addressRepo domain.CustomerAddressRepository
}

// NewProfileMergePreviewUseCase constructs a ProfileMergePreviewUseCase from
// the same profile/address repository interfaces ProfileMergeUseCase depends
// on (a strict subset — preview never touches demand/wave/fulfillment data).
func NewProfileMergePreviewUseCase(
	profileRepo domain.CustomerProfileRepository,
	addressRepo domain.CustomerAddressRepository,
) ProfileMergePreviewUseCase {
	return &profileMergePreviewUseCase{
		profileRepo: profileRepo,
		addressRepo: addressRepo,
	}
}

func (uc *profileMergePreviewUseCase) PreviewMergeProfiles(ctx context.Context, sourceProfileID, targetProfileID uint) (*dto.MergeProfilesPreviewResult, error) {
	if sourceProfileID == 0 || targetProfileID == 0 {
		return nil, fmt.Errorf("both source and target profile IDs are required")
	}
	if sourceProfileID == targetProfileID {
		return nil, fmt.Errorf("cannot preview a merge of a profile into itself")
	}

	src, err := uc.profileRepo.FindByID(ctx, sourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("source profile %d not found: %w", sourceProfileID, err)
	}
	tgt, err := uc.profileRepo.FindByID(ctx, targetProfileID)
	if err != nil {
		return nil, fmt.Errorf("target profile %d not found: %w", targetProfileID, err)
	}

	srcIdentities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, sourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("list source identities: %w", err)
	}
	tgtIdentities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, targetProfileID)
	if err != nil {
		return nil, fmt.Errorf("list target identities: %w", err)
	}
	srcAddresses, err := uc.addressRepo.ListByProfile(ctx, sourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("list source addresses: %w", err)
	}
	tgtAddresses, err := uc.addressRepo.ListByProfile(ctx, targetProfileID)
	if err != nil {
		return nil, fmt.Errorf("list target addresses: %w", err)
	}

	result := &dto.MergeProfilesPreviewResult{
		Source: dto.MergePreviewProfileSide{
			ProfileID:   src.ID,
			DisplayName: src.DisplayName,
			ProfileType: src.ProfileType,
			Identities:  identitiesToPreviewDTO(srcIdentities),
			Addresses:   addressesToPreviewDTO(srcAddresses),
		},
		Target: dto.MergePreviewProfileSide{
			ProfileID:   tgt.ID,
			DisplayName: tgt.DisplayName,
			ProfileType: tgt.ProfileType,
			Identities:  identitiesToPreviewDTO(tgtIdentities),
			Addresses:   addressesToPreviewDTO(tgtAddresses),
		},
		MovedIdentityCount: len(srcIdentities),
		MovedAddressCount:  len(srcAddresses),
	}

	// Conflicts: profile-level fields that disagree between source and
	// target. MergeProfiles keeps the target's own values and discards the
	// source's, so surfacing the diff up front is exactly what lets an
	// operator catch a wrong source/target pick before committing.
	if src.DisplayName != tgt.DisplayName {
		result.Conflicts = append(result.Conflicts, dto.MergePreviewConflict{
			Field:       "displayName",
			SourceValue: src.DisplayName,
			TargetValue: tgt.DisplayName,
		})
	}
	if src.ProfileType != tgt.ProfileType {
		result.Conflicts = append(result.Conflicts, dto.MergePreviewConflict{
			Field:       "profileType",
			SourceValue: src.ProfileType,
			TargetValue: tgt.ProfileType,
		})
	}

	// Duplicate identities: platform+value pairs present on BOTH sides would
	// collide once source identities are reassigned to the target profile.
	tgtIdentityKeys := make(map[string]struct{}, len(tgtIdentities))
	for _, ti := range tgtIdentities {
		tgtIdentityKeys[ti.IdentityPlatform+"::"+ti.IdentityValue] = struct{}{}
	}
	for _, si := range srcIdentities {
		key := si.IdentityPlatform + "::" + si.IdentityValue
		if _, dup := tgtIdentityKeys[key]; dup {
			result.DuplicateIdentityValues = append(result.DuplicateIdentityValues, key)
		}
	}

	return result, nil
}

func identitiesToPreviewDTO(identities []domain.CustomerIdentity) []dto.MergePreviewIdentity {
	out := make([]dto.MergePreviewIdentity, len(identities))
	for i, id := range identities {
		out[i] = dto.MergePreviewIdentity{
			ID:               id.ID,
			IdentityPlatform: id.IdentityPlatform,
			IdentityValue:    id.IdentityValue,
			IdentityType:     id.IdentityType,
			IsPrimary:        id.IsPrimary,
		}
	}
	return out
}

func addressesToPreviewDTO(addresses []domain.CustomerAddress) []dto.MergePreviewAddress {
	out := make([]dto.MergePreviewAddress, len(addresses))
	for i, a := range addresses {
		out[i] = dto.MergePreviewAddress{
			ID:            a.ID,
			Label:         a.Label,
			RecipientName: a.RecipientName,
			Phone:         a.Phone,
			Country:       a.Country,
			Province:      a.Province,
			City:          a.City,
			District:      a.District,
			AddressLine1:  a.AddressLine1,
			AddressLine2:  a.AddressLine2,
			PostalCode:    a.PostalCode,
			IsDefault:     a.IsDefault,
		}
	}
	return out
}
