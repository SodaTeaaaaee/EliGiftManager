package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// CarrierMappingUseCase manages carrier code translations for integration profiles.
type CarrierMappingUseCase interface {
	CreateMapping(input dto.CreateCarrierMappingInput) (*dto.CarrierMappingDTO, error)
	ListMappingsByProfile(profileID uint) ([]dto.CarrierMappingDTO, error)
	ResolveCarrier(profileID uint, internalCode string) (externalCode string, externalName string, err error)
	DeleteMapping(id uint) error
}

type carrierMappingUseCase struct {
	mappingRepo domain.CarrierMappingRepository
	profileRepo domain.IntegrationProfileRepository
}

// NewCarrierMappingUseCase returns a CarrierMappingUseCase.
func NewCarrierMappingUseCase(
	mappingRepo domain.CarrierMappingRepository,
	profileRepo domain.IntegrationProfileRepository,
) CarrierMappingUseCase {
	return &carrierMappingUseCase{
		mappingRepo: mappingRepo,
		profileRepo: profileRepo,
	}
}

func (uc *carrierMappingUseCase) CreateMapping(input dto.CreateCarrierMappingInput) (*dto.CarrierMappingDTO, error) {
	if input.IntegrationProfileID == 0 {
		return nil, fmt.Errorf("create carrier mapping: integrationProfileId is required")
	}
	if input.InternalCarrierCode == "" {
		return nil, fmt.Errorf("create carrier mapping: internalCarrierCode must not be empty")
	}
	if input.ExternalCarrierCode == "" {
		return nil, fmt.Errorf("create carrier mapping: externalCarrierCode must not be empty")
	}

	// Validate profile exists.
	if _, err := uc.profileRepo.FindByID(input.IntegrationProfileID); err != nil {
		return nil, fmt.Errorf("create carrier mapping: integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	mapping := &domain.CarrierMapping{
		IntegrationProfileID: input.IntegrationProfileID,
		InternalCarrierCode:  input.InternalCarrierCode,
		ExternalCarrierCode:  input.ExternalCarrierCode,
		ExternalCarrierName:  input.ExternalCarrierName,
		IsDefault:            input.IsDefault,
	}
	if err := uc.mappingRepo.Create(mapping); err != nil {
		return nil, fmt.Errorf("create carrier mapping: %w", err)
	}
	result := toCarrierMappingDTO(mapping)
	return &result, nil
}

func (uc *carrierMappingUseCase) ListMappingsByProfile(profileID uint) ([]dto.CarrierMappingDTO, error) {
	mappings, err := uc.mappingRepo.ListByProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("list carrier mappings for profile %d: %w", profileID, err)
	}
	result := make([]dto.CarrierMappingDTO, len(mappings))
	for i, m := range mappings {
		result[i] = toCarrierMappingDTO(&m)
	}
	return result, nil
}

func (uc *carrierMappingUseCase) ResolveCarrier(profileID uint, internalCode string) (string, string, error) {
	if internalCode == "" {
		return "", "", fmt.Errorf("resolve carrier: internalCode must not be empty")
	}
	mapping, err := uc.mappingRepo.FindByProfileAndInternal(profileID, internalCode)
	if err != nil {
		return "", "", fmt.Errorf("resolve carrier %q for profile %d: %w", internalCode, profileID, err)
	}
	return mapping.ExternalCarrierCode, mapping.ExternalCarrierName, nil
}

func (uc *carrierMappingUseCase) DeleteMapping(id uint) error {
	if id == 0 {
		return fmt.Errorf("delete carrier mapping: id is required")
	}
	return uc.mappingRepo.Delete(id)
}

func toCarrierMappingDTO(m *domain.CarrierMapping) dto.CarrierMappingDTO {
	return dto.CarrierMappingDTO{
		ID:                   m.ID,
		IntegrationProfileID: m.IntegrationProfileID,
		InternalCarrierCode:  m.InternalCarrierCode,
		ExternalCarrierCode:  m.ExternalCarrierCode,
		ExternalCarrierName:  m.ExternalCarrierName,
		IsDefault:            m.IsDefault,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}
