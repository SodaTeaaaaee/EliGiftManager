package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ProfileMergeUndoUseCase interface {
	UndoCustomerMerge(ctx context.Context, input dto.UndoCustomerMergeInput) (*dto.UndoCustomerMergeResult, error)
}

type profileMergeUndoUseCase struct {
	profileRepo domain.CustomerMergeProfileRepository
	addressRepo domain.CustomerMergeAddressRepository
	demandRepo  domain.CustomerMergeDemandRepository
	mergeRepo   domain.CustomerMergeRecordRepository
}

func NewProfileMergeUndoUseCase(
	profileRepo domain.CustomerMergeProfileRepository,
	addressRepo domain.CustomerMergeAddressRepository,
	demandRepo domain.CustomerMergeDemandRepository,
	mergeRepo domain.CustomerMergeRecordRepository,
) ProfileMergeUndoUseCase {
	return &profileMergeUndoUseCase{
		profileRepo: profileRepo,
		addressRepo: addressRepo,
		demandRepo:  demandRepo,
		mergeRepo:   mergeRepo,
	}
}

func (uc *profileMergeUndoUseCase) UndoCustomerMerge(ctx context.Context, input dto.UndoCustomerMergeInput) (*dto.UndoCustomerMergeResult, error) {
	if input.MergeID == 0 {
		return nil, fmt.Errorf("merge ID is required")
	}

	record, err := uc.mergeRepo.FindByID(ctx, input.MergeID)
	if err != nil {
		return nil, fmt.Errorf("merge record %d not found: %w", input.MergeID, err)
	}
	if record.UndoneAt != nil {
		return nil, fmt.Errorf("merge record %d has already been undone", record.ID)
	}

	var payload domain.CustomerMergePayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		return nil, fmt.Errorf("merge record %d has invalid payload: %w", record.ID, err)
	}
	if err := validateUniqueMergePayload(payload); err != nil {
		return nil, fmt.Errorf("merge record %d has invalid payload: %w", record.ID, err)
	}

	if _, err := uc.profileRepo.FindByID(ctx, record.TargetProfileID); err != nil {
		return nil, fmt.Errorf("target profile %d is missing or inactive: %w", record.TargetProfileID, err)
	}
	sourceDeleted, err := uc.profileRepo.IsSoftDeleted(ctx, record.SourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("source profile %d not found: %w", record.SourceProfileID, err)
	}
	if !sourceDeleted {
		return nil, fmt.Errorf("source profile %d is not soft-deleted", record.SourceProfileID)
	}

	identities, err := uc.profileRepo.ListIdentitiesByIDs(ctx, payload.IdentityIDs)
	if err != nil {
		return nil, fmt.Errorf("validate merged identities: %w", err)
	}
	identityOwners := make(map[uint]uint, len(identities))
	for _, identity := range identities {
		identityOwners[identity.ID] = identity.CustomerProfileID
	}
	if err := validateMergeRowOwners("identity", payload.IdentityIDs, identityOwners, record.TargetProfileID); err != nil {
		return nil, err
	}

	addresses, err := uc.addressRepo.ListByIDs(ctx, payload.AddressIDs)
	if err != nil {
		return nil, fmt.Errorf("validate merged addresses: %w", err)
	}
	addressOwners := make(map[uint]uint, len(addresses))
	for _, address := range addresses {
		addressOwners[address.ID] = address.CustomerProfileID
	}
	if err := validateMergeRowOwners("address", payload.AddressIDs, addressOwners, record.TargetProfileID); err != nil {
		return nil, err
	}

	documents, err := uc.demandRepo.ListByIDs(ctx, payload.DemandDocumentIDs)
	if err != nil {
		return nil, fmt.Errorf("validate merged demand documents: %w", err)
	}
	documentOwners := make(map[uint]uint, len(documents))
	for _, document := range documents {
		if document.CustomerProfileID != nil {
			documentOwners[document.ID] = *document.CustomerProfileID
		}
	}
	if err := validateMergeRowOwners("demand document", payload.DemandDocumentIDs, documentOwners, record.TargetProfileID); err != nil {
		return nil, err
	}

	if err := uc.profileRepo.RestoreSoftDeleted(ctx, record.SourceProfileID); err != nil {
		return nil, fmt.Errorf("restore source profile %d: %w", record.SourceProfileID, err)
	}
	if err := uc.profileRepo.BulkUpdateIdentityProfileID(ctx, payload.IdentityIDs, record.SourceProfileID); err != nil {
		return nil, fmt.Errorf("restore identities: %w", err)
	}
	if err := uc.addressRepo.BulkUpdateProfileIDByIDs(ctx, payload.AddressIDs, record.SourceProfileID); err != nil {
		return nil, fmt.Errorf("restore addresses: %w", err)
	}
	if err := uc.demandRepo.BulkUpdateCustomerProfileIDByIDs(ctx, payload.DemandDocumentIDs, record.SourceProfileID); err != nil {
		return nil, fmt.Errorf("restore demand documents: %w", err)
	}
	if err := uc.mergeRepo.MarkUndone(ctx, record.ID, time.Now()); err != nil {
		return nil, fmt.Errorf("mark merge record %d undone: %w", record.ID, err)
	}

	return &dto.UndoCustomerMergeResult{
		MergeID:                     record.ID,
		RestoredSourceProfileID:     record.SourceProfileID,
		TargetProfileID:             record.TargetProfileID,
		RestoredIdentityCount:       len(payload.IdentityIDs),
		RestoredAddressCount:        len(payload.AddressIDs),
		RestoredDemandDocumentCount: len(payload.DemandDocumentIDs),
	}, nil
}

func validateUniqueMergePayload(payload domain.CustomerMergePayload) error {
	groups := []struct {
		name string
		ids  []uint
	}{
		{name: "identity", ids: payload.IdentityIDs},
		{name: "address", ids: payload.AddressIDs},
		{name: "demand document", ids: payload.DemandDocumentIDs},
	}
	for _, group := range groups {
		seen := make(map[uint]struct{}, len(group.ids))
		for _, id := range group.ids {
			if id == 0 {
				return fmt.Errorf("%s ID must not be zero", group.name)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("duplicate %s ID %d", group.name, id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func validateMergeRowOwners(kind string, ids []uint, owners map[uint]uint, targetProfileID uint) error {
	for _, id := range ids {
		owner, exists := owners[id]
		if !exists {
			return fmt.Errorf("merged %s %d is missing", kind, id)
		}
		if owner != targetProfileID {
			return fmt.Errorf("merged %s %d belongs to profile %d, expected target profile %d", kind, id, owner, targetProfileID)
		}
	}
	return nil
}
