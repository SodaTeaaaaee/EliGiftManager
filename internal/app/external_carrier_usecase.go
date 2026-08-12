package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ExternalCarrierUseCase owns profile-scoped carrier discovery. It never
// performs cross-profile binding; only BindInternalCarrier may attach an
// internal canonical carrier code.
type ExternalCarrierUseCase struct {
	repo domain.ExternalCarrierRepository
}

// ExternalCarrierObservationInput is the internal observation command used by
// import pipelines. Provenance is intentionally kept out of the public
// RegisterExternalCarrier DTO so callers cannot forge RAW evidence links.
type ExternalCarrierObservationInput struct {
	IntegrationProfileID uint
	ExternalCarrierCode  string
	ExternalCarrierName  string
	SourceImportRunID    *uint
	SourceRawRecordID    *uint
}

func NewExternalCarrierUseCase(repo domain.ExternalCarrierRepository) *ExternalCarrierUseCase {
	return &ExternalCarrierUseCase{repo: repo}
}

func ExternalCarrierCanonicalKey(code, name string) (string, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code != "" {
		return "code:" + code, nil
	}
	name = normalizeExternalCarrierName(name)
	if name == "" {
		return "", fmt.Errorf("external carrier code or name is required")
	}
	return "name:" + name, nil
}
func normalizeExternalCarrierName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

func (uc *ExternalCarrierUseCase) RegisterExternalCarrier(ctx context.Context, input dto.RegisterExternalCarrierInput) (*dto.ExternalCarrierDTO, error) {
	return uc.registerExternalCarrier(ctx, ExternalCarrierObservationInput{
		IntegrationProfileID: input.IntegrationProfileID,
		ExternalCarrierCode:  input.ExternalCarrierCode,
		ExternalCarrierName:  input.ExternalCarrierName,
	}, true)
}

// ObserveExternalCarrier is idempotent for repeated exact observations (for
// example many shipment rows using the same carrier) while still surfacing
// code/name conflicts for review.
func (uc *ExternalCarrierUseCase) ObserveExternalCarrier(ctx context.Context, input dto.RegisterExternalCarrierInput) (*dto.ExternalCarrierDTO, error) {
	return uc.ObserveExternalCarrierWithProvenance(ctx, ExternalCarrierObservationInput{
		IntegrationProfileID: input.IntegrationProfileID,
		ExternalCarrierCode:  input.ExternalCarrierCode,
		ExternalCarrierName:  input.ExternalCarrierName,
	})
}

// ObserveExternalCarrierWithProvenance records the first durable RAW source for
// a newly discovered carrier. Exact later observations return the existing row
// without replacing its original provenance.
func (uc *ExternalCarrierUseCase) ObserveExternalCarrierWithProvenance(ctx context.Context, input ExternalCarrierObservationInput) (*dto.ExternalCarrierDTO, error) {
	return uc.registerExternalCarrier(ctx, input, false)
}

func (uc *ExternalCarrierUseCase) registerExternalCarrier(ctx context.Context, input ExternalCarrierObservationInput, duplicateRequiresReview bool) (*dto.ExternalCarrierDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureCarrierRegistry); err != nil {
		return nil, err
	}
	key, err := ExternalCarrierCanonicalKey(input.ExternalCarrierCode, input.ExternalCarrierName)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	all, err := uc.repo.ListByProfile(ctx, input.IntegrationProfileID)
	if err != nil {
		return nil, err
	}
	normalizedName := normalizeExternalCarrierName(input.ExternalCarrierName)
	for i := range all {
		if all[i].CanonicalKey != key && normalizedName != "" && normalizeExternalCarrierName(all[i].ExternalCarrierName) == normalizedName {
			reason := "same normalized external name observed with different external codes"
			all[i].Status = "review"
			all[i].ConflictReason = reason
			all[i].UpdatedAt = now
			if err := uc.repo.Update(ctx, &all[i]); err != nil {
				return nil, err
			}
			if err := uc.repo.CreateConflict(ctx, &domain.ExternalCarrierConflict{IntegrationProfileID: input.IntegrationProfileID, CanonicalKey: key, ConflictKind: "same_name_different_code", ExternalCarrierCode: input.ExternalCarrierCode, ExternalCarrierName: input.ExternalCarrierName, SourceImportRunID: copyOptionalUint(input.SourceImportRunID), SourceRawRecordID: copyOptionalUint(input.SourceRawRecordID), CreatedAt: now}); err != nil {
				return nil, err
			}
		}
	}
	existing, err := uc.repo.FindByCanonicalKey(ctx, input.IntegrationProfileID, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !duplicateRequiresReview && strings.EqualFold(strings.TrimSpace(existing.ExternalCarrierCode), strings.TrimSpace(input.ExternalCarrierCode)) && strings.EqualFold(strings.TrimSpace(existing.ExternalCarrierName), strings.TrimSpace(input.ExternalCarrierName)) {
			out := externalCarrierDTO(existing)
			return &out, nil
		}
		kind := "duplicate_in_file"
		reason := "duplicate external carrier observation requires review"
		if strings.TrimSpace(existing.ExternalCarrierName) != strings.TrimSpace(input.ExternalCarrierName) {
			kind = "same_code_different_name"
			reason = "same external carrier code observed with a different name"
		}
		existing.Status = "review"
		existing.ConflictReason = reason
		existing.UpdatedAt = now
		if err := uc.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		if err := uc.repo.CreateConflict(ctx, &domain.ExternalCarrierConflict{IntegrationProfileID: input.IntegrationProfileID, CanonicalKey: key, ConflictKind: kind, ExternalCarrierCode: input.ExternalCarrierCode, ExternalCarrierName: input.ExternalCarrierName, SourceImportRunID: copyOptionalUint(input.SourceImportRunID), SourceRawRecordID: copyOptionalUint(input.SourceRawRecordID), CreatedAt: now}); err != nil {
			return nil, err
		}
		out := externalCarrierDTO(existing)
		return &out, nil
	}
	carrier := &domain.ExternalCarrier{IntegrationProfileID: input.IntegrationProfileID, CanonicalKey: key, ExternalCarrierCode: strings.TrimSpace(input.ExternalCarrierCode), ExternalCarrierName: strings.TrimSpace(input.ExternalCarrierName), NameKeyStrategy: "code_or_normalized_name_v1", Status: "provisional", SourceImportRunID: copyOptionalUint(input.SourceImportRunID), SourceRawRecordID: copyOptionalUint(input.SourceRawRecordID), CreatedAt: now, UpdatedAt: now}
	if err := uc.repo.Create(ctx, carrier); err != nil {
		return nil, err
	}
	out := externalCarrierDTO(carrier)
	return &out, nil
}

func copyOptionalUint(value *uint) *uint {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (uc *ExternalCarrierUseCase) BindInternalCarrier(ctx context.Context, input dto.BindInternalCarrierInput) (*dto.ExternalCarrierDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureCarrierRegistry); err != nil {
		return nil, err
	}
	code := strings.TrimSpace(input.InternalCarrierCode)
	if code == "" {
		return nil, fmt.Errorf("internalCarrierCode is required")
	}
	carrier, err := uc.repo.FindByID(ctx, input.ExternalCarrierID)
	if err != nil {
		return nil, err
	}
	if carrier == nil {
		return nil, fmt.Errorf("external carrier %d not found", input.ExternalCarrierID)
	}
	carrier.InternalCarrierCode = &code
	carrier.Status = "bound"
	carrier.ConflictReason = ""
	carrier.UpdatedAt = time.Now().UTC()
	if err := uc.repo.Update(ctx, carrier); err != nil {
		return nil, err
	}
	out := externalCarrierDTO(carrier)
	return &out, nil
}
func (uc *ExternalCarrierUseCase) ListByProfile(ctx context.Context, profileID uint) ([]dto.ExternalCarrierDTO, error) {
	rows, err := uc.repo.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ExternalCarrierDTO, len(rows))
	for i := range rows {
		out[i] = externalCarrierDTO(&rows[i])
	}
	return out, nil
}

func (uc *ExternalCarrierUseCase) listDomainByProfile(ctx context.Context, profileID uint) ([]domain.ExternalCarrier, error) {
	return uc.repo.ListByProfile(ctx, profileID)
}

func (uc *ExternalCarrierUseCase) RecordConflicts(ctx context.Context, conflicts []domain.ExternalCarrierConflict) error {
	if len(conflicts) == 0 {
		return nil
	}
	return uc.repo.CreateConflicts(ctx, conflicts)
}

func (uc *ExternalCarrierUseCase) MarkReview(ctx context.Context, id uint, reason string) error {
	carrier, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if carrier == nil {
		return fmt.Errorf("external carrier %d not found", id)
	}
	carrier.Status = "review"
	carrier.ConflictReason = reason
	carrier.UpdatedAt = time.Now().UTC()
	return uc.repo.Update(ctx, carrier)
}
func externalCarrierDTO(c *domain.ExternalCarrier) dto.ExternalCarrierDTO {
	return dto.ExternalCarrierDTO{ID: c.ID, IntegrationProfileID: c.IntegrationProfileID, CanonicalKey: c.CanonicalKey, ExternalCarrierCode: c.ExternalCarrierCode, ExternalCarrierName: c.ExternalCarrierName, NameKeyStrategy: c.NameKeyStrategy, InternalCarrierCode: c.InternalCarrierCode, Status: c.Status, ConflictReason: c.ConflictReason, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}
