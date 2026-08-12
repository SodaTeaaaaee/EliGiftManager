package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// AddressBatchUseCase provides batch address-binding operations for fulfillment lines,
// composing over the same repositories AddressManagementUseCase uses.
type AddressBatchUseCase interface {
	// BatchBindAddressToLines binds each entry's address to its fulfillment line, returning
	// a per-entry result (partial-success semantics — one failing entry does not abort the rest).
	BatchBindAddressToLines(ctx context.Context, entries []dto.BindAddressEntry) ([]dto.AddressBatchItemResult, error)

	// BindDefaultAddressesForWave binds the recipient's default address to every
	// address-missing fulfillment line in the given wave, returning a per-line result.
	BindDefaultAddressesForWave(ctx context.Context, waveID uint) ([]dto.AddressBatchItemResult, error)
}

type addressBatchUseCase struct {
	addressRepo     domain.CustomerAddressRepository
	fulfillmentRepo domain.FulfillmentLineRepository
}

// NewAddressBatchUseCase constructs an AddressBatchUseCase.
func NewAddressBatchUseCase(
	addressRepo domain.CustomerAddressRepository,
	fulfillmentRepo domain.FulfillmentLineRepository,
) AddressBatchUseCase {
	return &addressBatchUseCase{
		addressRepo:     addressRepo,
		fulfillmentRepo: fulfillmentRepo,
	}
}

// bindOne mirrors addressManagementUseCase.BindAddressToLine but reports failures as a
// result item instead of aborting the whole batch.
func (uc *addressBatchUseCase) bindOne(ctx context.Context, lineID, addressID uint) dto.AddressBatchItemResult {
	addr, err := uc.addressRepo.FindByID(ctx, addressID)
	if err != nil {
		return dto.AddressBatchItemResult{
			FulfillmentLineID: lineID,
			Success:           false,
			ErrorMessage:      "address not found: " + err.Error(),
		}
	}

	line, err := uc.fulfillmentRepo.FindByID(ctx, lineID)
	if err != nil {
		return dto.AddressBatchItemResult{
			FulfillmentLineID: lineID,
			Success:           false,
			ErrorMessage:      "fulfillment line not found: " + err.Error(),
		}
	}

	line.CustomerAddressID = &addr.ID
	line.AddressState = deriveAddressState(addr.ValidationStatus)

	if err := uc.fulfillmentRepo.Update(ctx, line); err != nil {
		return dto.AddressBatchItemResult{
			FulfillmentLineID: lineID,
			Success:           false,
			ErrorMessage:      "bind address to line: " + err.Error(),
		}
	}

	boundID := addr.ID
	return dto.AddressBatchItemResult{
		FulfillmentLineID: lineID,
		CustomerAddressID: &boundID,
		Success:           true,
	}
}

func (uc *addressBatchUseCase) BatchBindAddressToLines(ctx context.Context, entries []dto.BindAddressEntry) ([]dto.AddressBatchItemResult, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	results := make([]dto.AddressBatchItemResult, 0, len(entries))
	for _, entry := range entries {
		results = append(results, uc.bindOne(ctx, entry.FulfillmentLineID, entry.CustomerAddressID))
	}
	return results, nil
}

func (uc *addressBatchUseCase) BindDefaultAddressesForWave(ctx context.Context, waveID uint) ([]dto.AddressBatchItemResult, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	lines, err := uc.fulfillmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, err
	}

	results := make([]dto.AddressBatchItemResult, 0)
	// Cache the resolved default address per customer profile so a wave with many lines
	// for the same recipient only looks it up once.
	defaultCache := make(map[uint]*domain.CustomerAddress)

	for i := range lines {
		line := lines[i]

		if domain.AddressState(line.AddressState) != domain.AddressStateMissing {
			continue
		}

		if line.CustomerProfileID == nil {
			results = append(results, dto.AddressBatchItemResult{
				FulfillmentLineID: line.ID,
				Success:           false,
				ErrorMessage:      "fulfillment line has no customer profile",
			})
			continue
		}
		profileID := *line.CustomerProfileID

		defaultAddr, cached := defaultCache[profileID]
		if !cached {
			addrs, err := uc.addressRepo.ListByProfile(ctx, profileID)
			if err != nil {
				results = append(results, dto.AddressBatchItemResult{
					FulfillmentLineID: line.ID,
					Success:           false,
					ErrorMessage:      "list addresses: " + err.Error(),
				})
				continue
			}
			for j := range addrs {
				if addrs[j].IsDefault {
					found := addrs[j]
					defaultAddr = &found
					break
				}
			}
			defaultCache[profileID] = defaultAddr
		}

		if defaultAddr == nil {
			results = append(results, dto.AddressBatchItemResult{
				FulfillmentLineID: line.ID,
				Success:           false,
				ErrorMessage:      "no default address for customer profile",
			})
			continue
		}

		results = append(results, uc.bindOne(ctx, line.ID, defaultAddr.ID))
	}

	return results, nil
}
