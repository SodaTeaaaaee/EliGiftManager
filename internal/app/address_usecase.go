package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type addressManagementUseCase struct {
	addressRepo     domain.CustomerAddressRepository
	fulfillmentRepo domain.FulfillmentLineRepository
}

func NewAddressManagementUseCase(
	addressRepo domain.CustomerAddressRepository,
	fulfillmentRepo domain.FulfillmentLineRepository,
) AddressManagementUseCase {
	return &addressManagementUseCase{
		addressRepo:     addressRepo,
		fulfillmentRepo: fulfillmentRepo,
	}
}

func (uc *addressManagementUseCase) CreateAddress(ctx context.Context, input dto.CreateAddressInput) (*dto.CustomerAddressDTO, error) {
	if input.Label == "" {
		return nil, fmt.Errorf("address label is required")
	}

	if input.IsDefault {
		if err := uc.addressRepo.ClearDefaultByProfile(ctx, input.CustomerProfileID); err != nil {
			return nil, fmt.Errorf("clear existing defaults: %w", err)
		}
	}

	status := input.ValidationStatus
	if status == "" {
		status = "unvalidated"
	}

	addr := &domain.CustomerAddress{
		CustomerProfileID: input.CustomerProfileID,
		Label:             input.Label,
		RecipientName:     input.RecipientName,
		Phone:             input.Phone,
		Country:           input.Country,
		Province:          input.Province,
		City:              input.City,
		District:          input.District,
		AddressLine1:      input.AddressLine1,
		AddressLine2:      input.AddressLine2,
		PostalCode:        input.PostalCode,
		IsDefault:         input.IsDefault,
		IsTest:            input.IsTest,
		ValidationStatus:  status,
		ValidationDetail:  input.ValidationDetail,
		ExtraData:         input.ExtraData,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := uc.addressRepo.Create(ctx, addr); err != nil {
		return nil, err
	}
	result := addressToDTO(addr)
	return &result, nil
}

func (uc *addressManagementUseCase) UpdateAddress(ctx context.Context, input dto.UpdateAddressInput) (*dto.CustomerAddressDTO, error) {
	existing, err := uc.addressRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Label != "" {
		existing.Label = input.Label
	}
	if input.IsDefault && !existing.IsDefault {
		if err := uc.addressRepo.ClearDefaultByProfile(ctx, input.CustomerProfileID); err != nil {
			return nil, fmt.Errorf("clear existing defaults: %w", err)
		}
	}

	existing.CustomerProfileID = input.CustomerProfileID
	existing.RecipientName = input.RecipientName
	existing.Phone = input.Phone
	existing.Country = input.Country
	existing.Province = input.Province
	existing.City = input.City
	existing.District = input.District
	existing.AddressLine1 = input.AddressLine1
	existing.AddressLine2 = input.AddressLine2
	existing.PostalCode = input.PostalCode
	existing.IsDefault = input.IsDefault
	existing.IsTest = input.IsTest
	existing.ValidationStatus = input.ValidationStatus
	existing.ValidationDetail = input.ValidationDetail
	existing.ExtraData = input.ExtraData
	existing.UpdatedAt = time.Now()

	if err := uc.addressRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	result := addressToDTO(existing)
	return &result, nil
}

func (uc *addressManagementUseCase) DeleteAddress(ctx context.Context, id uint) error {
	return uc.addressRepo.SoftDelete(ctx, id)
}

func (uc *addressManagementUseCase) GetAddress(ctx context.Context, id uint) (*dto.CustomerAddressDTO, error) {
	addr, err := uc.addressRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := addressToDTO(addr)
	return &result, nil
}

func (uc *addressManagementUseCase) ListAddressesByProfile(ctx context.Context, profileID uint) ([]dto.CustomerAddressDTO, error) {
	addrs, err := uc.addressRepo.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CustomerAddressDTO, len(addrs))
	for i := range addrs {
		result[i] = addressToDTO(&addrs[i])
	}
	return result, nil
}

func (uc *addressManagementUseCase) BindAddressToLine(ctx context.Context, input dto.BindAddressInput) (*dto.CustomerAddressDTO, error) {
	addr, err := uc.addressRepo.FindByID(ctx, input.CustomerAddressID)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}

	line, err := uc.fulfillmentRepo.FindByID(ctx, input.FulfillmentLineID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment line not found: %w", err)
	}

	line.CustomerAddressID = &addr.ID
	line.AddressState = deriveAddressState(addr.ValidationStatus)

	if err := uc.fulfillmentRepo.Update(ctx, line); err != nil {
		return nil, fmt.Errorf("bind address to line: %w", err)
	}

	result := addressToDTO(addr)
	return &result, nil
}

func (uc *addressManagementUseCase) UnbindAddressFromLine(ctx context.Context, fulfillmentLineID uint) error {
	line, err := uc.fulfillmentRepo.FindByID(ctx, fulfillmentLineID)
	if err != nil {
		return fmt.Errorf("fulfillment line not found: %w", err)
	}

	line.CustomerAddressID = nil
	line.AddressState = "missing"

	return uc.fulfillmentRepo.Update(ctx, line)
}

func deriveAddressState(validationStatus string) string {
	switch validationStatus {
	case "valid":
		return "ready"
	case "invalid":
		return "invalid"
	default:
		return "ready"
	}
}

func addressToDTO(a *domain.CustomerAddress) dto.CustomerAddressDTO {
	return dto.CustomerAddressDTO{
		ID:                a.ID,
		CustomerProfileID: a.CustomerProfileID,
		Label:             a.Label,
		RecipientName:     a.RecipientName,
		Phone:             a.Phone,
		Country:           a.Country,
		Province:          a.Province,
		City:              a.City,
		District:          a.District,
		AddressLine1:      a.AddressLine1,
		AddressLine2:      a.AddressLine2,
		PostalCode:        a.PostalCode,
		IsDefault:         a.IsDefault,
		IsTest:            a.IsTest,
		ValidationStatus:  a.ValidationStatus,
		ValidationDetail:  a.ValidationDetail,
		ExtraData:         a.ExtraData,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}
