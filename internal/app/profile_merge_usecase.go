package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type profileMergeUseCase struct {
	profileRepo     domain.CustomerProfileRepository
	addressRepo     domain.CustomerAddressRepository
	demandRepo      domain.DemandDocumentRepository
	waveRepo        domain.WaveRepository
	fulfillmentRepo domain.FulfillmentLineRepository
}

func NewProfileMergeUseCase(
	profileRepo domain.CustomerProfileRepository,
	addressRepo domain.CustomerAddressRepository,
	demandRepo domain.DemandDocumentRepository,
	waveRepo domain.WaveRepository,
	fulfillmentRepo domain.FulfillmentLineRepository,
) ProfileMergeUseCase {
	return &profileMergeUseCase{
		profileRepo:     profileRepo,
		addressRepo:     addressRepo,
		demandRepo:      demandRepo,
		waveRepo:        waveRepo,
		fulfillmentRepo: fulfillmentRepo,
	}
}

func (uc *profileMergeUseCase) MergeProfiles(input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error) {
	if input.SourceProfileID == 0 || input.TargetProfileID == 0 {
		return nil, fmt.Errorf("both source and target profile IDs are required")
	}
	if input.SourceProfileID == input.TargetProfileID {
		return nil, fmt.Errorf("cannot merge a profile into itself")
	}

	// Verify both profiles exist
	src, err := uc.profileRepo.FindByID(input.SourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("source profile %d not found: %w", input.SourceProfileID, err)
	}
	_, err = uc.profileRepo.FindByID(input.TargetProfileID)
	if err != nil {
		return nil, fmt.Errorf("target profile %d not found: %w", input.TargetProfileID, err)
	}

	result := &dto.MergeProfilesResult{}

	// 1. Migrate identities
	identities, err := uc.profileRepo.ListIdentitiesByProfile(src.ID)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	identityIDs := make([]uint, len(identities))
	for i := range identities {
		identityIDs[i] = identities[i].ID
	}
	if len(identityIDs) > 0 {
		if err := uc.profileRepo.BulkUpdateIdentityProfileID(identityIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("migrate identities: %w", err)
		}
		result.MigratedIdentityCount = len(identityIDs)
	}

	// 2. Migrate addresses — count source addresses before migration so the
	// result reflects only the rows actually moved (not pre-existing target rows).
	srcAddrs, err := uc.addressRepo.ListByProfile(src.ID)
	if err != nil {
		return nil, fmt.Errorf("list source addresses: %w", err)
	}
	if err := uc.addressRepo.BulkUpdateProfileID(src.ID, input.TargetProfileID); err != nil {
		return nil, fmt.Errorf("migrate addresses: %w", err)
	}
	result.MigratedAddressCount = len(srcAddrs)

	// 3. Reassign demand documents
	demandCount, err := uc.demandRepo.BulkUpdateCustomerProfileID(src.ID, input.TargetProfileID)
	if err != nil {
		return nil, fmt.Errorf("reassign demand documents: %w", err)
	}
	result.UpdatedDemandDocs = int(demandCount)

	// 4. Reassign wave participant snapshots
	participantCount, err := uc.waveRepo.UpdateParticipantProfileID(src.ID, input.TargetProfileID)
	if err != nil {
		return nil, fmt.Errorf("reassign participants: %w", err)
	}
	result.UpdatedParticipants = int(participantCount)

	// 5. Reassign fulfillment lines
	fulfillCount, err := uc.fulfillmentRepo.BulkUpdateCustomerProfileID(src.ID, input.TargetProfileID)
	if err != nil {
		return nil, fmt.Errorf("reassign fulfillment lines: %w", err)
	}
	result.UpdatedFulfillmentLines = int(fulfillCount)

	// 6. Soft-delete the source profile
	if err := uc.profileRepo.SoftDelete(src.ID); err != nil {
		return nil, fmt.Errorf("soft-delete source profile: %w", err)
	}

	return result, nil
}
