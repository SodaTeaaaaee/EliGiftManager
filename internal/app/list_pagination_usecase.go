package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ListPaginationUseCase struct {
	customers domain.CustomerProfilePageRepository
	products  domain.ProductMasterPageRepository
	demand    domain.DemandInboxPageRepository
	shipments domain.ShipmentPageRepository
	waves     domain.WaveRepository
	profiles  domain.IntegrationProfileRepository
}

func NewListPaginationUseCase(
	customers domain.CustomerProfilePageRepository,
	products domain.ProductMasterPageRepository,
	demand domain.DemandInboxPageRepository,
	shipments domain.ShipmentPageRepository,
	waves domain.WaveRepository,
	profiles domain.IntegrationProfileRepository,
) *ListPaginationUseCase {
	return &ListPaginationUseCase{
		customers: customers,
		products:  products,
		demand:    demand,
		shipments: shipments,
		waves:     waves,
		profiles:  profiles,
	}
}

func (uc *ListPaginationUseCase) ListCustomerProfilesPage(ctx context.Context, input dto.CustomerProfilePageFilterInput) (dto.CustomerProfilePageResult, error) {
	input.Limit, input.Offset = dto.NormalizeListPagination(input.Limit, input.Offset)
	profiles, total, err := uc.customers.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{
		ListPageQuery: domain.ListPageQuery{SortBy: input.SortBy, SortDir: input.SortDir, Limit: input.Limit, Offset: input.Offset},
		Keyword:       input.Keyword, Platform: input.Platform, MissingAddressOnly: input.MissingAddressOnly,
	})
	if err != nil {
		return dto.CustomerProfilePageResult{}, err
	}
	ids := make([]uint, len(profiles))
	for i := range profiles {
		ids[i] = profiles[i].ID
	}
	identities, err := uc.customers.ListCustomerIdentitiesByProfileIDs(ctx, ids)
	if err != nil {
		return dto.CustomerProfilePageResult{}, err
	}
	addresses, err := uc.customers.ListCustomerAddressesByProfileIDs(ctx, ids)
	if err != nil {
		return dto.CustomerProfilePageResult{}, err
	}
	matchedNames, err := uc.customers.FindMatchedCustomerHistoricalNames(ctx, ids, input.Keyword)
	if err != nil {
		return dto.CustomerProfilePageResult{}, err
	}
	identitiesByProfile := make(map[uint][]domain.CustomerIdentity, len(ids))
	for _, identity := range identities {
		identitiesByProfile[identity.CustomerProfileID] = append(identitiesByProfile[identity.CustomerProfileID], identity)
	}
	addressesByProfile := make(map[uint][]domain.CustomerAddress, len(ids))
	for _, address := range addresses {
		addressesByProfile[address.CustomerProfileID] = append(addressesByProfile[address.CustomerProfileID], address)
	}
	items := make([]dto.CustomerProfileDTO, len(profiles))
	for i := range profiles {
		items[i] = customerProfilePageDTO(profiles[i], identitiesByProfile[profiles[i].ID], addressesByProfile[profiles[i].ID])
		items[i].MatchedHistoricalName = matchedNames[profiles[i].ID]
	}
	return dto.CustomerProfilePageResult{Items: items, TotalCount: int(total)}, nil
}

func (uc *ListPaginationUseCase) ListCustomerIdentityPlatforms(ctx context.Context) ([]string, error) {
	return uc.customers.ListCustomerIdentityPlatforms(ctx)
}

func (uc *ListPaginationUseCase) ListProductMastersPage(ctx context.Context, input dto.ProductMasterPageFilterInput) (dto.ProductMasterPageResult, error) {
	input.Limit, input.Offset = dto.NormalizeListPagination(input.Limit, input.Offset)
	masters, total, err := uc.products.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{
		ListPageQuery: domain.ListPageQuery{SortBy: input.SortBy, SortDir: input.SortDir, Limit: input.Limit, Offset: input.Offset},
		Keyword:       input.Keyword, ProductKinds: input.ProductKinds, ArchivedOnly: input.ArchivedOnly,
	})
	if err != nil {
		return dto.ProductMasterPageResult{}, err
	}
	items := make([]dto.ProductMasterDTO, len(masters))
	for i := range masters {
		items[i] = productMasterToDTO(&masters[i])
	}
	return dto.ProductMasterPageResult{Items: items, TotalCount: int(total)}, nil
}

func (uc *ListPaginationUseCase) ListDemandInboxRowsPage(ctx context.Context, input dto.DemandInboxFilterInput) (dto.DemandInboxPageResult, error) {
	input.Limit, input.Offset = dto.NormalizeListPagination(input.Limit, input.Offset)
	docs, total, err := uc.demand.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		ListPageQuery: domain.ListPageQuery{SortBy: input.SortBy, SortDir: input.SortDir, Limit: input.Limit, Offset: input.Offset},
		Assignment:    input.Assignment, DemandKind: input.DemandKind, IntegrationProfileID: input.IntegrationProfileID, WaveID: input.WaveID,
	})
	if err != nil {
		return dto.DemandInboxPageResult{}, err
	}
	ids := make([]uint, len(docs))
	for i := range docs {
		ids[i] = docs[i].ID
	}
	assignments, err := uc.demand.ListDemandAssignmentsByDocumentIDs(ctx, ids)
	if err != nil {
		return dto.DemandInboxPageResult{}, err
	}
	lines, err := uc.demand.ListDemandLinesByDocumentIDs(ctx, ids)
	if err != nil {
		return dto.DemandInboxPageResult{}, err
	}
	waves, err := uc.waves.List(ctx)
	if err != nil {
		return dto.DemandInboxPageResult{}, err
	}
	profiles, err := uc.profiles.List(ctx)
	if err != nil {
		return dto.DemandInboxPageResult{}, err
	}
	return dto.DemandInboxPageResult{Items: AssembleDemandInboxRows(docs, assignments, lines, waves, profiles), TotalCount: int(total)}, nil
}

func (uc *ListPaginationUseCase) ListShipmentsByWavePage(ctx context.Context, input dto.ShipmentByWavePageFilterInput) (dto.ShipmentPageResult, error) {
	input.Limit, input.Offset = dto.NormalizeListPagination(input.Limit, input.Offset)
	shipments, total, err := uc.shipments.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{
		ListPageQuery: domain.ListPageQuery{SortBy: input.SortBy, SortDir: input.SortDir, Limit: input.Limit, Offset: input.Offset},
		WaveID:        input.WaveID,
	})
	if err != nil {
		return dto.ShipmentPageResult{}, err
	}
	ids := make([]uint, len(shipments))
	for i := range shipments {
		ids[i] = shipments[i].ID
	}
	lines, err := uc.shipments.ListShipmentLinesByShipmentIDs(ctx, ids)
	if err != nil {
		return dto.ShipmentPageResult{}, err
	}
	linesByShipment := make(map[uint][]domain.ShipmentLine, len(ids))
	for _, line := range lines {
		linesByShipment[line.ShipmentID] = append(linesByShipment[line.ShipmentID], line)
	}
	items := make([]dto.ShipmentDTO, len(shipments))
	for i := range shipments {
		items[i] = shipmentPageDTO(shipments[i], linesByShipment[shipments[i].ID])
	}
	return dto.ShipmentPageResult{Items: items, TotalCount: int(total)}, nil
}

func AssembleDemandInboxRows(docs []domain.DemandDocument, assignments []domain.WaveDemandAssignment, lines []domain.DemandLine, waves []domain.Wave, profiles []domain.IntegrationProfile) []dto.DemandInboxRowDTO {
	assignmentsByDoc := make(map[uint][]domain.WaveDemandAssignment)
	for _, assignment := range assignments {
		assignmentsByDoc[assignment.DemandDocumentID] = append(assignmentsByDoc[assignment.DemandDocumentID], assignment)
	}
	linesByDoc := make(map[uint][]domain.DemandLine)
	for _, line := range lines {
		linesByDoc[line.DemandDocumentID] = append(linesByDoc[line.DemandDocumentID], line)
	}
	waveMap := make(map[uint]domain.Wave, len(waves))
	for _, wave := range waves {
		waveMap[wave.ID] = wave
	}
	profileMap := make(map[uint]domain.IntegrationProfile, len(profiles))
	for _, profile := range profiles {
		profileMap[profile.ID] = profile
	}
	rows := make([]dto.DemandInboxRowDTO, 0, len(docs))
	for _, doc := range docs {
		docAssignments := assignmentsByDoc[doc.ID]
		row := dto.DemandInboxRowDTO{
			DemandDocumentID: doc.ID, Kind: doc.Kind, CaptureMode: doc.CaptureMode,
			SourceChannel: doc.SourceChannel, SourceSurface: doc.SourceSurface, SourceDocumentNo: doc.SourceDocumentNo,
			CustomerProfileID: doc.CustomerProfileID, IntegrationProfileID: doc.IntegrationProfileID,
			Assigned: len(docAssignments) > 0, CreatedAt: doc.CreatedAt,
		}
		if doc.IntegrationProfileID != nil {
			if profile, ok := profileMap[*doc.IntegrationProfileID]; ok {
				row.IntegrationProfileLabel = fmt.Sprintf("%s (%s)", profile.ProfileKey, profile.SourceChannel)
			}
		}
		if row.Assigned {
			waveID := docAssignments[0].WaveID
			row.AssignedWaveID = &waveID
			if wave, ok := waveMap[waveID]; ok {
				row.AssignedWaveLabel = fmt.Sprintf("%s \u2014 %s", wave.WaveNo, wave.Name)
			}
		}
		for _, line := range linesByDoc[doc.ID] {
			row.TotalLineCount++
			switch line.RoutingDisposition {
			case "accepted":
				row.AcceptedCount++
				if line.RecipientInputState == "ready" || line.RecipientInputState == "not_required" {
					row.ReadyAcceptedCount++
				}
				if line.RecipientInputState == "waiting_for_input" || line.RecipientInputState == "partially_collected" {
					row.WaitingInputCount++
				}
			case "deferred":
				row.DeferredCount++
			case "excluded_manual", "excluded_duplicate", "excluded_revoked":
				row.ExcludedCount++
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func customerProfilePageDTO(profile domain.CustomerProfile, identities []domain.CustomerIdentity, addresses []domain.CustomerAddress) dto.CustomerProfileDTO {
	identityDTOs := make([]dto.CustomerIdentityDTO, len(identities))
	for i, identity := range identities {
		identityDTOs[i] = dto.CustomerIdentityDTO{ID: identity.ID, CustomerProfileID: identity.CustomerProfileID, IdentityPlatform: identity.IdentityPlatform, IdentityValue: identity.IdentityValue, IdentityType: identity.IdentityType, IsPrimary: identity.IsPrimary, ExtraData: identity.ExtraData, CreatedAt: identity.CreatedAt, UpdatedAt: identity.UpdatedAt}
	}
	addressDTOs := make([]dto.CustomerAddressDTO, len(addresses))
	for i, address := range addresses {
		addressDTOs[i] = dto.CustomerAddressDTO{ID: address.ID, CustomerProfileID: address.CustomerProfileID, Label: address.Label, RecipientName: address.RecipientName, Phone: address.Phone, Country: address.Country, Province: address.Province, City: address.City, District: address.District, AddressLine1: address.AddressLine1, AddressLine2: address.AddressLine2, PostalCode: address.PostalCode, IsDefault: address.IsDefault, IsTest: address.IsTest, ValidationStatus: address.ValidationStatus, ValidationDetail: address.ValidationDetail, ExtraData: address.ExtraData, CreatedAt: address.CreatedAt, UpdatedAt: address.UpdatedAt}
	}
	return dto.CustomerProfileDTO{ID: profile.ID, DisplayName: profile.DisplayName, ProfileType: profile.ProfileType,
		Status: profile.Status, MergedIntoProfileID: profile.MergedIntoProfileID, RowVersion: profile.RowVersion,
		DisplayNameMode: profile.DisplayNameMode, DisplayNameObservationID: profile.DisplayNameObservationID,
		ExtraData: profile.ExtraData, CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt,
		Identities: identityDTOs, Addresses: addressDTOs, ActiveAddressCount: len(addresses)}
}

func shipmentPageDTO(shipment domain.Shipment, lines []domain.ShipmentLine) dto.ShipmentDTO {
	lineDTOs := make([]dto.ShipmentLineDTO, len(lines))
	for i, line := range lines {
		lineDTOs[i] = dto.ShipmentLineDTO{ID: line.ID, ShipmentID: line.ShipmentID, SupplierOrderLineID: line.SupplierOrderLineID, FulfillmentLineID: line.FulfillmentLineID, Quantity: line.Quantity, CreatedAt: line.CreatedAt}
	}
	return dto.ShipmentDTO{ID: shipment.ID, SupplierOrderID: shipment.SupplierOrderID, SupplierPlatform: shipment.SupplierPlatform, ShipmentNo: shipment.ShipmentNo, ExternalShipmentNo: shipment.ExternalShipmentNo, CarrierCode: shipment.CarrierCode, CarrierName: shipment.CarrierName, TrackingNo: shipment.TrackingNo, Status: shipment.Status, ShippedAt: shipment.ShippedAt, BasisHistoryNodeID: shipment.BasisHistoryNodeID, BasisProjectionHash: shipment.BasisProjectionHash, BasisPayloadSnapshot: shipment.BasisPayloadSnapshot, ExtraData: shipment.ExtraData, CreatedAt: shipment.CreatedAt, UpdatedAt: shipment.UpdatedAt, Lines: lineDTOs}
}
