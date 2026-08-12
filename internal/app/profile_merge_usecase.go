package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type profileMergeUseCase struct {
	profileRepo domain.CustomerMergeProfileRepository
	addressRepo domain.CustomerMergeAddressRepository
	demandRepo  domain.CustomerMergeDemandRepository
	mergeRepo   domain.CustomerMergeRecordRepository
	nameRepo    domain.CustomerNameObservationRepository
	originRepo  domain.CustomerProfileOriginRepository
}

type CustomerMergeResolutionRepos struct {
	NameObservations domain.CustomerNameObservationRepository
	Origins          domain.CustomerProfileOriginRepository
}

func NewProfileMergeUseCase(
	profileRepo domain.CustomerMergeProfileRepository,
	addressRepo domain.CustomerMergeAddressRepository,
	demandRepo domain.CustomerMergeDemandRepository,
	mergeRepo domain.CustomerMergeRecordRepository,
	resolution ...CustomerMergeResolutionRepos,
) ProfileMergeUseCase {
	uc := &profileMergeUseCase{
		profileRepo: profileRepo,
		addressRepo: addressRepo,
		demandRepo:  demandRepo,
		mergeRepo:   mergeRepo,
	}
	if len(resolution) > 0 {
		uc.nameRepo = resolution[0].NameObservations
		uc.originRepo = resolution[0].Origins
	}
	return uc
}

func (uc *profileMergeUseCase) MergeProfiles(ctx context.Context, input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.profileRepo, domain.CustomerResolutionFeatureMergeExecution); err != nil {
		return nil, err
	}
	if input.SourceProfileID == 0 || input.TargetProfileID == 0 {
		return nil, fmt.Errorf("both source and target profile IDs are required")
	}
	if input.SourceProfileID == input.TargetProfileID {
		return nil, fmt.Errorf("cannot merge a profile into itself")
	}

	// Verify both profiles exist and are active.
	src, err := uc.profileRepo.FindByID(ctx, input.SourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("source profile %d not found: %w", input.SourceProfileID, err)
	}
	_, err = uc.profileRepo.FindByID(ctx, input.TargetProfileID)
	if err != nil {
		return nil, fmt.Errorf("target profile %d not found: %w", input.TargetProfileID, err)
	}

	result := &dto.MergeProfilesResult{}

	// 1. Migrate identities
	identities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, src.ID)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	identityIDs := make([]uint, len(identities))
	for i := range identities {
		identityIDs[i] = identities[i].ID
	}
	if len(identityIDs) > 0 {
		if err := uc.profileRepo.BulkUpdateIdentityProfileID(ctx, identityIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("migrate identities: %w", err)
		}
		result.MigratedIdentityCount = len(identityIDs)
	}

	// 2. Migrate addresses by the exact IDs captured before mutation.
	srcAddrs, err := uc.addressRepo.ListByProfile(ctx, src.ID)
	if err != nil {
		return nil, fmt.Errorf("list source addresses: %w", err)
	}
	addressIDs := make([]uint, len(srcAddrs))
	for i := range srcAddrs {
		addressIDs[i] = srcAddrs[i].ID
	}
	if len(addressIDs) > 0 {
		if err := uc.addressRepo.BulkUpdateProfileIDByIDs(ctx, addressIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("migrate addresses: %w", err)
		}
	}
	result.MigratedAddressCount = len(srcAddrs)

	// 3. Reassign only demand documents that have never been assigned to a wave.
	documents, err := uc.demandRepo.ListUnassignedByCustomerProfileID(ctx, src.ID)
	if err != nil {
		return nil, fmt.Errorf("list unassigned demand documents: %w", err)
	}
	documentIDs := make([]uint, len(documents))
	for i := range documents {
		documentIDs[i] = documents[i].ID
	}
	if len(documentIDs) > 0 {
		if err := uc.demandRepo.BulkUpdateCustomerProfileIDByIDs(ctx, documentIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("reassign demand documents: %w", err)
		}
	}
	result.UpdatedDemandDocs = len(documentIDs)

	var nameObservationIDs []uint
	if uc.nameRepo != nil {
		observations, err := uc.nameRepo.ListByProfile(ctx, src.ID)
		if err != nil {
			return nil, fmt.Errorf("list source name observations: %w", err)
		}
		nameObservationIDs = make([]uint, len(observations))
		for i := range observations {
			nameObservationIDs[i] = observations[i].ID
		}
		if err := uc.nameRepo.BulkUpdateProfileIDByIDs(ctx, nameObservationIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("migrate name observations: %w", err)
		}
	}

	var originIDs []uint
	if uc.originRepo != nil {
		origins, err := uc.originRepo.ListByProfile(ctx, src.ID)
		if err != nil {
			return nil, fmt.Errorf("list source customer origins: %w", err)
		}
		originIDs = make([]uint, len(origins))
		for i := range origins {
			originIDs[i] = origins[i].ID
		}
		if err := uc.originRepo.BulkUpdateProfileIDByIDs(ctx, originIDs, input.TargetProfileID); err != nil {
			return nil, fmt.Errorf("migrate customer origins: %w", err)
		}
	}

	// Historical participant snapshots and fulfillment lines intentionally stay
	// attached to the source profile.

	// 4. Soft-delete the source profile.
	if err := uc.profileRepo.SoftDelete(ctx, src.ID); err != nil {
		return nil, fmt.Errorf("soft-delete source profile: %w", err)
	}

	payload, err := json.Marshal(domain.CustomerMergePayload{
		IdentityIDs:        identityIDs,
		AddressIDs:         addressIDs,
		DemandDocumentIDs:  documentIDs,
		NameObservationIDs: nameObservationIDs,
		OriginIDs:          originIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("encode merge payload: %w", err)
	}
	record := &domain.CustomerMergeRecord{
		SourceProfileID: src.ID,
		TargetProfileID: input.TargetProfileID,
		Payload:         string(payload),
		CreatedAt:       time.Now(),
	}
	if err := uc.mergeRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create merge record: %w", err)
	}
	result.MergeID = record.ID
	result.UndoAvailable = true

	return result, nil
}
