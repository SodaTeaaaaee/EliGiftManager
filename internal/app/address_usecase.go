package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type addressManagementUseCase struct {
	addressRepo     domain.CustomerAddressRepository
	fulfillmentRepo domain.FulfillmentLineRepository
	db              *gorm.DB
}

func NewAddressManagementUseCase(
	addressRepo domain.CustomerAddressRepository,
	fulfillmentRepo domain.FulfillmentLineRepository,
	extra ...any,
) AddressManagementUseCase {
	uc := &addressManagementUseCase{
		addressRepo:     addressRepo,
		fulfillmentRepo: fulfillmentRepo,
	}
	for _, dep := range extra {
		if db, ok := dep.(*gorm.DB); ok && db != nil {
			uc.db = db
		}
	}
	return uc
}

// addressDefaultMutator is the ClearDefault + persist surface wrapped in a transaction.
type addressDefaultMutator interface {
	ClearDefaultByProfile(ctx context.Context, profileID uint) error
	Create(ctx context.Context, addr *domain.CustomerAddress) error
	Update(ctx context.Context, addr *domain.CustomerAddress) error
}

// gormAddressWriter rebinds address writes onto a transaction (or nested savepoint) handle.
type gormAddressWriter struct {
	db *gorm.DB
}

func (w *gormAddressWriter) ClearDefaultByProfile(ctx context.Context, profileID uint) error {
	return w.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).
		Where("customer_profile_id = ? AND is_default = ?", profileID, true).
		Update("is_default", false).Error
}

func (w *gormAddressWriter) Create(ctx context.Context, addr *domain.CustomerAddress) error {
	p := persistence.CustomerAddressFromDomain(addr)
	if err := w.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*addr = *persistence.CustomerAddressToDomain(p)
	return nil
}

func (w *gormAddressWriter) Update(ctx context.Context, addr *domain.CustomerAddress) error {
	p := persistence.CustomerAddressFromDomain(addr)
	p.ID = addr.ID
	return w.db.WithContext(ctx).Save(p).Error
}

func (uc *addressManagementUseCase) persistAfterClearDefault(
	ctx context.Context,
	profileID uint,
	addr *domain.CustomerAddress,
	create bool,
) error {
	write := func(repo addressDefaultMutator) error {
		if err := repo.ClearDefaultByProfile(ctx, profileID); err != nil {
			return fmt.Errorf("clear existing defaults: %w", err)
		}
		if create {
			return repo.Create(ctx, addr)
		}
		return repo.Update(ctx, addr)
	}
	if uc.db == nil {
		return write(uc.addressRepo)
	}
	// GORM uses a savepoint when db is already a transaction, so controller
	// (or import) outer txs are not double-committed as independent sessions.
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return write(&gormAddressWriter{db: tx})
	})
}

func (uc *addressManagementUseCase) CreateAddress(ctx context.Context, input dto.CreateAddressInput) (*dto.CustomerAddressDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	if input.CustomerProfileID == 0 {
		return nil, fmt.Errorf("customer profile ID is required")
	}
	if input.Label == "" {
		return nil, fmt.Errorf("address label is required")
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

	if input.IsDefault {
		if err := uc.persistAfterClearDefault(ctx, input.CustomerProfileID, addr, true); err != nil {
			return nil, err
		}
	} else if err := uc.addressRepo.Create(ctx, addr); err != nil {
		return nil, err
	}
	result := addressToDTO(addr)
	return &result, nil
}

func (uc *addressManagementUseCase) UpdateAddress(ctx context.Context, input dto.UpdateAddressInput) (*dto.CustomerAddressDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	existing, err := uc.addressRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.CustomerProfileID != 0 {
		if existing.CustomerProfileID != 0 && existing.CustomerProfileID != input.CustomerProfileID {
			return nil, fmt.Errorf("cannot reassign address %d from profile %d to profile %d", existing.ID, existing.CustomerProfileID, input.CustomerProfileID)
		}
		existing.CustomerProfileID = input.CustomerProfileID
	}

	if input.Label != "" {
		existing.Label = input.Label
	}

	existing.RecipientName = input.RecipientName
	existing.Phone = input.Phone
	existing.Country = input.Country
	existing.Province = input.Province
	existing.City = input.City
	existing.District = input.District
	existing.AddressLine1 = input.AddressLine1
	existing.AddressLine2 = input.AddressLine2
	existing.PostalCode = input.PostalCode
	existing.IsTest = input.IsTest
	existing.ValidationStatus = input.ValidationStatus
	existing.ValidationDetail = input.ValidationDetail
	existing.ExtraData = input.ExtraData
	existing.UpdatedAt = time.Now()

	promoteDefault := input.IsDefault && !existing.IsDefault
	existing.IsDefault = input.IsDefault
	if promoteDefault {
		if err := uc.persistAfterClearDefault(ctx, existing.CustomerProfileID, existing, false); err != nil {
			return nil, err
		}
	} else if err := uc.addressRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	result := addressToDTO(existing)
	return &result, nil
}

func (uc *addressManagementUseCase) DeleteAddress(ctx context.Context, id uint) error {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return err
	}
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
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	addr, err := uc.addressRepo.FindByID(ctx, input.CustomerAddressID)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}

	line, err := uc.fulfillmentRepo.FindByID(ctx, input.FulfillmentLineID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment line not found: %w", err)
	}

	if line.CustomerProfileID == nil || addr.CustomerProfileID != *line.CustomerProfileID {
		return nil, fmt.Errorf("cannot bind address to fulfillment line: customer profile mismatch")
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
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return err
	}
	line, err := uc.fulfillmentRepo.FindByID(ctx, fulfillmentLineID)
	if err != nil {
		return fmt.Errorf("fulfillment line not found: %w", err)
	}

	line.CustomerAddressID = nil
	line.AddressState = "missing"

	return uc.fulfillmentRepo.Update(ctx, line)
}

// UpsertAddressFromImport hybrid-matches an existing address on the customer profile:
//   - primary key: RecipientName + Phone
//   - fallback when Phone is empty: RecipientName + AddressLine1
//
// Hit → update non-empty draft fields onto the existing row.
// Miss → create a new address (label defaults to "import"; IsDefault honouring draft).
func (uc *addressManagementUseCase) UpsertAddressFromImport(ctx context.Context, customerProfileID uint, draft RecipientAddressDraft) (*dto.CustomerAddressDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.addressRepo, domain.CustomerResolutionFeatureWrites); err != nil {
		return nil, err
	}
	if customerProfileID == 0 {
		return nil, fmt.Errorf("upsert address from import: customerProfileID is required")
	}
	if draft.RecipientName == "" {
		return nil, fmt.Errorf("upsert address from import: recipient name is required")
	}

	addrs, err := uc.addressRepo.ListByProfile(ctx, customerProfileID)
	if err != nil {
		return nil, fmt.Errorf("list addresses for profile %d: %w", customerProfileID, err)
	}

	var match *domain.CustomerAddress
	for i := range addrs {
		a := &addrs[i]
		if draft.Phone != "" {
			if a.RecipientName == draft.RecipientName && a.Phone == draft.Phone {
				match = a
				break
			}
			continue
		}
		// Phone empty → Name + AddressLine1
		if a.RecipientName == draft.RecipientName && a.AddressLine1 == draft.AddressLine1 {
			match = a
			break
		}
	}

	now := time.Now()
	if match != nil {
		applyRecipientDraft(match, draft)
		match.UpdatedAt = now
		if draft.IsDefault && !match.IsDefault {
			match.IsDefault = true
			if err := uc.persistAfterClearDefault(ctx, customerProfileID, match, false); err != nil {
				return nil, err
			}
		} else if err := uc.addressRepo.Update(ctx, match); err != nil {
			return nil, fmt.Errorf("update matched address: %w", err)
		}
		result := addressToDTO(match)
		return &result, nil
	}

	// Create path.
	label := draft.Label
	if label == "" {
		label = "import"
	}
	// First address for a profile becomes default when the draft does not opt out.
	isDefault := draft.IsDefault
	if !isDefault && len(addrs) == 0 {
		isDefault = true
	}

	addr := &domain.CustomerAddress{
		CustomerProfileID: customerProfileID,
		Label:             label,
		RecipientName:     draft.RecipientName,
		Phone:             draft.Phone,
		Country:           draft.Country,
		Province:          draft.Province,
		City:              draft.City,
		District:          draft.District,
		AddressLine1:      draft.AddressLine1,
		AddressLine2:      draft.AddressLine2,
		PostalCode:        draft.PostalCode,
		IsDefault:         isDefault,
		ValidationStatus:  "unvalidated",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if draft.IsDefault {
		if err := uc.persistAfterClearDefault(ctx, customerProfileID, addr, true); err != nil {
			return nil, err
		}
	} else if err := uc.addressRepo.Create(ctx, addr); err != nil {
		return nil, fmt.Errorf("create address from import: %w", err)
	}
	result := addressToDTO(addr)
	return &result, nil
}

// applyRecipientDraft copies non-empty draft fields onto an existing address.
func applyRecipientDraft(addr *domain.CustomerAddress, draft RecipientAddressDraft) {
	if draft.RecipientName != "" {
		addr.RecipientName = draft.RecipientName
	}
	if draft.Phone != "" {
		addr.Phone = draft.Phone
	}
	if draft.Country != "" {
		addr.Country = draft.Country
	}
	if draft.Province != "" {
		addr.Province = draft.Province
	}
	if draft.City != "" {
		addr.City = draft.City
	}
	if draft.District != "" {
		addr.District = draft.District
	}
	if draft.AddressLine1 != "" {
		addr.AddressLine1 = draft.AddressLine1
	}
	if draft.AddressLine2 != "" {
		addr.AddressLine2 = draft.AddressLine2
	}
	if draft.PostalCode != "" {
		addr.PostalCode = draft.PostalCode
	}
	if draft.Label != "" {
		addr.Label = draft.Label
	}
}

func deriveAddressState(validationStatus string) string {
	switch domain.AddressValidationStatus(validationStatus) {
	case domain.AddressValidationStatusValid:
		return string(domain.AddressStateReady)
	case domain.AddressValidationStatusInvalid:
		return string(domain.AddressStateInvalid)
	default:
		return string(domain.AddressStateMissing)
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
