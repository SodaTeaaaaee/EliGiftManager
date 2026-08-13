package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

// ---- CustomerProfile ----

func CustomerProfileFromDomain(d *domain.CustomerProfile) *CustomerProfile {
	return &CustomerProfile{
		DisplayName:              d.DisplayName,
		ProfileType:              ProfileType(d.ProfileType),
		Status:                   d.Status,
		MergedIntoProfileID:      d.MergedIntoProfileID,
		RowVersion:               d.RowVersion,
		DisplayNameMode:          d.DisplayNameMode,
		DisplayNameObservationID: d.DisplayNameObservationID,
		ExtraData:                d.ExtraData,
	}
}

func CustomerProfileToDomain(p *CustomerProfile) *domain.CustomerProfile {
	return &domain.CustomerProfile{
		ID:                       p.ID,
		DisplayName:              p.DisplayName,
		ProfileType:              string(p.ProfileType),
		Status:                   p.Status,
		MergedIntoProfileID:      p.MergedIntoProfileID,
		RowVersion:               p.RowVersion,
		DisplayNameMode:          p.DisplayNameMode,
		DisplayNameObservationID: p.DisplayNameObservationID,
		ExtraData:                p.ExtraData,
		CreatedAt:                p.CreatedAt,
		UpdatedAt:                p.UpdatedAt,
	}
}

// ---- CustomerIdentity ----

func CustomerIdentityFromDomain(d *domain.CustomerIdentity) *CustomerIdentity {
	return &CustomerIdentity{
		CustomerProfileID:          d.CustomerProfileID,
		IdentityPlatform:           d.IdentityPlatform,
		IdentityValue:              d.IdentityValue,
		IdentityType:               IdentityType(d.IdentityType),
		Namespace:                  d.Namespace,
		NormalizedValue:            d.NormalizedValue,
		NormalizationVersion:       d.NormalizationVersion,
		Authority:                  d.Authority,
		VerificationStatus:         d.VerificationStatus,
		SourceIntegrationProfileID: d.SourceIntegrationProfileID,
		ResolutionStatus:           d.ResolutionStatus,
		FirstSeenAt:                d.FirstSeenAt,
		LastSeenAt:                 d.LastSeenAt,
		IsPrimary:                  d.IsPrimary,
		ExtraData:                  d.ExtraData,
	}
}

func CustomerIdentityToDomain(p *CustomerIdentity) *domain.CustomerIdentity {
	return &domain.CustomerIdentity{
		ID:                         p.ID,
		CustomerProfileID:          p.CustomerProfileID,
		IdentityPlatform:           p.IdentityPlatform,
		IdentityValue:              p.IdentityValue,
		IdentityType:               string(p.IdentityType),
		Namespace:                  p.Namespace,
		NormalizedValue:            p.NormalizedValue,
		NormalizationVersion:       p.NormalizationVersion,
		Authority:                  p.Authority,
		VerificationStatus:         p.VerificationStatus,
		SourceIntegrationProfileID: p.SourceIntegrationProfileID,
		ResolutionStatus:           p.ResolutionStatus,
		FirstSeenAt:                p.FirstSeenAt,
		LastSeenAt:                 p.LastSeenAt,
		IsPrimary:                  p.IsPrimary,
		ExtraData:                  p.ExtraData,
		CreatedAt:                  p.CreatedAt,
		UpdatedAt:                  p.UpdatedAt,
	}
}

// ---- CustomerAddress ----

func CustomerAddressFromDomain(d *domain.CustomerAddress) *CustomerAddress {
	return &CustomerAddress{
		CustomerProfileID:    d.CustomerProfileID,
		Label:                d.Label,
		RecipientName:        d.RecipientName,
		Phone:                d.Phone,
		NormalizedPhone:      d.NormalizedPhone,
		AddressFingerprint:   d.AddressFingerprint,
		NormalizationVersion: d.NormalizationVersion,
		QualityStatus:        d.QualityStatus,
		Country:              d.Country,
		Province:             d.Province,
		City:                 d.City,
		District:             d.District,
		AddressLine1:         d.AddressLine1,
		AddressLine2:         d.AddressLine2,
		PostalCode:           d.PostalCode,
		IsDefault:            d.IsDefault,
		IsTest:               d.IsTest,
		ValidationStatus:     AddressValidationStatus(d.ValidationStatus),
		ValidationDetail:     d.ValidationDetail,
		ExtraData:            d.ExtraData,
	}
}

func CustomerAddressToDomain(p *CustomerAddress) *domain.CustomerAddress {
	return &domain.CustomerAddress{
		ID:                   p.ID,
		CustomerProfileID:    p.CustomerProfileID,
		Label:                p.Label,
		RecipientName:        p.RecipientName,
		Phone:                p.Phone,
		NormalizedPhone:      p.NormalizedPhone,
		AddressFingerprint:   p.AddressFingerprint,
		NormalizationVersion: p.NormalizationVersion,
		QualityStatus:        p.QualityStatus,
		Country:              p.Country,
		Province:             p.Province,
		City:                 p.City,
		District:             p.District,
		AddressLine1:         p.AddressLine1,
		AddressLine2:         p.AddressLine2,
		PostalCode:           p.PostalCode,
		IsDefault:            p.IsDefault,
		IsTest:               p.IsTest,
		ValidationStatus:     string(p.ValidationStatus),
		ValidationDetail:     p.ValidationDetail,
		ExtraData:            p.ExtraData,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ---- DemandDocument ----

func DemandDocumentFromDomain(d *domain.DemandDocument) *DemandDocument {
	return &DemandDocument{
		Kind:                 DemandKind(d.Kind),
		CaptureMode:          CaptureMode(d.CaptureMode),
		SourceChannel:        d.SourceChannel,
		SourceSurface:        d.SourceSurface,
		IntegrationProfileID: d.IntegrationProfileID,
		SourceDocumentNo:     d.SourceDocumentNo,
		SourceCustomerRef:    d.SourceCustomerRef,
		CustomerProfileID:    d.CustomerProfileID,
		SourceCreatedAt:      d.SourceCreatedAt,
		SourcePaidAt:         d.SourcePaidAt,
		Currency:             d.Currency,
		AuthoritySnapshotAt:  d.AuthoritySnapshotAt,
		RawPayload:           d.RawPayload,
		ExtraData:            d.ExtraData,
		BoundProfileSnapshot: d.BoundProfileSnapshot,
	}
}

func DemandDocumentToDomain(p *DemandDocument) *domain.DemandDocument {
	return &domain.DemandDocument{
		ID:                   p.ID,
		Kind:                 string(p.Kind),
		CaptureMode:          string(p.CaptureMode),
		SourceChannel:        p.SourceChannel,
		SourceSurface:        p.SourceSurface,
		IntegrationProfileID: p.IntegrationProfileID,
		SourceDocumentNo:     p.SourceDocumentNo,
		SourceCustomerRef:    p.SourceCustomerRef,
		CustomerProfileID:    p.CustomerProfileID,
		SourceCreatedAt:      p.SourceCreatedAt,
		SourcePaidAt:         p.SourcePaidAt,
		Currency:             p.Currency,
		AuthoritySnapshotAt:  p.AuthoritySnapshotAt,
		RawPayload:           p.RawPayload,
		ExtraData:            p.ExtraData,
		BoundProfileSnapshot: p.BoundProfileSnapshot,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ---- DemandLine ----

func DemandLineFromDomain(d *domain.DemandLine) *DemandLine {
	return &DemandLine{
		DemandDocumentID:      d.DemandDocumentID,
		SourceLineNo:          d.SourceLineNo,
		LineType:              DemandLineType(d.LineType),
		ObligationTriggerKind: ObligationTriggerKind(d.ObligationTriggerKind),
		EntitlementAuthority:  EntitlementAuthority(d.EntitlementAuthority),
		RecipientInputState:   RecipientInputState(d.RecipientInputState),
		RoutingDisposition:    RoutingDisposition(d.RoutingDisposition),
		RoutingReasonCode:     d.RoutingReasonCode,
		EligibilityContextRef: d.EligibilityContextRef,
		ProductMasterID:       d.ProductMasterID,
		ExternalTitle:         d.ExternalTitle,
		RequestedQuantity:     d.RequestedQuantity,
		EntitlementCode:       d.EntitlementCode,
		GiftLevelSnapshot:     d.GiftLevelSnapshot,
		RecipientInputPayload: d.RecipientInputPayload,
		RawPayload:            d.RawPayload,
		ExtraData:             d.ExtraData,
	}
}

func DemandLineToDomain(p *DemandLine) *domain.DemandLine {
	return &domain.DemandLine{
		ID:                    p.ID,
		DemandDocumentID:      p.DemandDocumentID,
		SourceLineNo:          p.SourceLineNo,
		LineType:              string(p.LineType),
		ObligationTriggerKind: string(p.ObligationTriggerKind),
		EntitlementAuthority:  string(p.EntitlementAuthority),
		RecipientInputState:   string(p.RecipientInputState),
		RoutingDisposition:    string(p.RoutingDisposition),
		RoutingReasonCode:     p.RoutingReasonCode,
		EligibilityContextRef: p.EligibilityContextRef,
		ProductMasterID:       p.ProductMasterID,
		ExternalTitle:         p.ExternalTitle,
		RequestedQuantity:     p.RequestedQuantity,
		EntitlementCode:       p.EntitlementCode,
		GiftLevelSnapshot:     p.GiftLevelSnapshot,
		RecipientInputPayload: p.RecipientInputPayload,
		RawPayload:            p.RawPayload,
		ExtraData:             p.ExtraData,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// ---- Wave ----

func WaveFromDomain(d *domain.Wave) *Wave {
	return &Wave{
		WaveNo:           d.WaveNo,
		Name:             d.Name,
		WaveType:         WaveType(d.WaveType),
		LifecycleStage:   d.LifecycleStage,
		ProgressSnapshot: d.ProgressSnapshot,
		Notes:            d.Notes,
		LevelTags:        d.LevelTags,
	}
}

func WaveToDomain(p *Wave) *domain.Wave {
	return &domain.Wave{
		ID:               p.ID,
		WaveNo:           p.WaveNo,
		Name:             p.Name,
		WaveType:         string(p.WaveType),
		LifecycleStage:   p.LifecycleStage,
		ProgressSnapshot: p.ProgressSnapshot,
		Notes:            p.Notes,
		LevelTags:        p.LevelTags,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// ---- WaveParticipantSnapshot ----

func WaveParticipantSnapshotFromDomain(d *domain.WaveParticipantSnapshot) *WaveParticipantSnapshot {
	return &WaveParticipantSnapshot{
		ID:                 d.ID,
		WaveID:             d.WaveID,
		CustomerProfileID:  d.CustomerProfileID,
		SnapshotType:       SnapshotType(d.SnapshotType),
		IdentityPlatform:   d.IdentityPlatform,
		IdentityValue:      d.IdentityValue,
		DisplayName:        d.DisplayName,
		GiftLevel:          d.GiftLevel,
		SourceDocumentRefs: d.SourceDocumentRefs,
		SourceProfileRefs:  d.SourceProfileRefs,
		ExtraData:          d.ExtraData,
	}
}

func WaveParticipantSnapshotToDomain(p *WaveParticipantSnapshot) *domain.WaveParticipantSnapshot {
	return &domain.WaveParticipantSnapshot{
		ID:                 p.ID,
		WaveID:             p.WaveID,
		CustomerProfileID:  p.CustomerProfileID,
		SnapshotType:       string(p.SnapshotType),
		IdentityPlatform:   p.IdentityPlatform,
		IdentityValue:      p.IdentityValue,
		DisplayName:        p.DisplayName,
		GiftLevel:          p.GiftLevel,
		SourceDocumentRefs: p.SourceDocumentRefs,
		SourceProfileRefs:  p.SourceProfileRefs,
		ExtraData:          p.ExtraData,
		CreatedAt:          p.CreatedAt,
	}
}

// ---- FulfillmentLine ----

func FulfillmentLineFromDomain(d *domain.FulfillmentLine) *FulfillmentLine {
	return &FulfillmentLine{
		WaveID:                    d.WaveID,
		CustomerProfileID:         d.CustomerProfileID,
		WaveParticipantSnapshotID: d.WaveParticipantSnapshotID,
		ProductID:                 d.ProductID,
		DemandDocumentID:          d.DemandDocumentID,
		DemandLineID:              d.DemandLineID,
		CustomerAddressID:         d.CustomerAddressID,
		Quantity:                  d.Quantity,
		AllocationState:           d.AllocationState,
		AddressState:              d.AddressState,
		SupplierState:             d.SupplierState,
		ChannelSyncState:          d.ChannelSyncState,
		LineReason:                FulfillmentLineReason(d.LineReason),
		GeneratedBy:               d.GeneratedBy,
		ExtraData:                 d.ExtraData,
	}
}

func FulfillmentLineToDomain(p *FulfillmentLine) *domain.FulfillmentLine {
	return &domain.FulfillmentLine{
		ID:                        p.ID,
		WaveID:                    p.WaveID,
		CustomerProfileID:         p.CustomerProfileID,
		WaveParticipantSnapshotID: p.WaveParticipantSnapshotID,
		ProductID:                 p.ProductID,
		DemandDocumentID:          p.DemandDocumentID,
		DemandLineID:              p.DemandLineID,
		CustomerAddressID:         p.CustomerAddressID,
		Quantity:                  p.Quantity,
		AllocationState:           p.AllocationState,
		AddressState:              p.AddressState,
		SupplierState:             p.SupplierState,
		ChannelSyncState:          p.ChannelSyncState,
		LineReason:                string(p.LineReason),
		GeneratedBy:               p.GeneratedBy,
		ExtraData:                 p.ExtraData,
		CreatedAt:                 p.CreatedAt,
		UpdatedAt:                 p.UpdatedAt,
	}
}

// ---- AllocationPolicyRule ----

func AllocationPolicyRuleFromDomain(d *domain.AllocationPolicyRule) (*AllocationPolicyRule, error) {
	selectorJSON, err := json.Marshal(d.SelectorPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal selector payload: %w", err)
	}
	return &AllocationPolicyRule{
		WaveID:               d.WaveID,
		ProductID:            d.ProductID,
		SelectorPayload:      string(selectorJSON),
		ProductTargetRef:     d.ProductTargetRef,
		ContributionQuantity: d.ContributionQuantity,
		RuleKind:             d.RuleKind,
		Priority:             d.Priority,
		Active:               d.Active,
	}, nil
}

func AllocationPolicyRuleToDomain(p *AllocationPolicyRule) (*domain.AllocationPolicyRule, error) {
	var selector domain.SelectorPayload
	if p.SelectorPayload != "" {
		if err := json.Unmarshal([]byte(p.SelectorPayload), &selector); err != nil {
			return nil, fmt.Errorf("unmarshal selector payload: %w", err)
		}
	}
	return &domain.AllocationPolicyRule{
		ID:                   p.ID,
		WaveID:               p.WaveID,
		ProductID:            p.ProductID,
		SelectorPayload:      selector,
		ProductTargetRef:     p.ProductTargetRef,
		ContributionQuantity: p.ContributionQuantity,
		RuleKind:             p.RuleKind,
		Priority:             p.Priority,
		Active:               p.Active,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}, nil
}

// ---- SupplierOrder ----

func SupplierOrderFromDomain(d *domain.SupplierOrder) *SupplierOrder {
	return &SupplierOrder{
		WaveID:                      d.WaveID,
		FactoryIntegrationProfileID: d.FactoryIntegrationProfileID,
		SupplierPlatform:            d.SupplierPlatform,
		TemplateID:                  d.TemplateID,
		BatchNo:                     d.BatchNo,
		ExternalOrderNo:             d.ExternalOrderNo,
		SubmissionMode:              SubmissionMode(d.SubmissionMode),
		SubmittedAt:                 d.SubmittedAt,
		Status:                      SupplierOrderStatus(d.Status),
		RequestPayload:              d.RequestPayload,
		ResponsePayload:             d.ResponsePayload,
		BasisHistoryNodeID:          d.BasisHistoryNodeID,
		BasisProjectionHash:         d.BasisProjectionHash,
		BasisPayloadSnapshot:        d.BasisPayloadSnapshot,
		ExtraData:                   d.ExtraData,
	}
}

func SupplierOrderToDomain(p *SupplierOrder) *domain.SupplierOrder {
	return &domain.SupplierOrder{
		ID:                          p.ID,
		WaveID:                      p.WaveID,
		FactoryIntegrationProfileID: p.FactoryIntegrationProfileID,
		SupplierPlatform:            p.SupplierPlatform,
		TemplateID:                  p.TemplateID,
		BatchNo:                     p.BatchNo,
		ExternalOrderNo:             p.ExternalOrderNo,
		SubmissionMode:              string(p.SubmissionMode),
		SubmittedAt:                 p.SubmittedAt,
		Status:                      string(p.Status),
		RequestPayload:              p.RequestPayload,
		ResponsePayload:             p.ResponsePayload,
		BasisHistoryNodeID:          p.BasisHistoryNodeID,
		BasisProjectionHash:         p.BasisProjectionHash,
		BasisPayloadSnapshot:        p.BasisPayloadSnapshot,
		ExtraData:                   p.ExtraData,
		CreatedAt:                   p.CreatedAt,
		UpdatedAt:                   p.UpdatedAt,
	}
}

// ---- SupplierOrderLine ----

func SupplierOrderLineFromDomain(d *domain.SupplierOrderLine) *SupplierOrderLine {
	return &SupplierOrderLine{
		SupplierOrderID:   d.SupplierOrderID,
		FulfillmentLineID: d.FulfillmentLineID,
		SupplierLineNo:    d.SupplierLineNo,
		SupplierSKU:       d.SupplierSKU,
		SubmittedQuantity: d.SubmittedQuantity,
		AcceptedQuantity:  d.AcceptedQuantity,
		Status:            d.Status,
		ExtraData:         d.ExtraData,
	}
}

func SupplierOrderLineToDomain(p *SupplierOrderLine) *domain.SupplierOrderLine {
	return &domain.SupplierOrderLine{
		ID:                p.ID,
		SupplierOrderID:   p.SupplierOrderID,
		FulfillmentLineID: p.FulfillmentLineID,
		SupplierLineNo:    p.SupplierLineNo,
		SupplierSKU:       p.SupplierSKU,
		SubmittedQuantity: p.SubmittedQuantity,
		AcceptedQuantity:  p.AcceptedQuantity,
		Status:            p.Status,
		ExtraData:         p.ExtraData,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

// ---- WaveDemandAssignment ----

func WaveDemandAssignmentFromDomain(d *domain.WaveDemandAssignment) *WaveDemandAssignment {
	return &WaveDemandAssignment{
		WaveID:           d.WaveID,
		DemandDocumentID: d.DemandDocumentID,
		AcceptedAt:       d.AcceptedAt,
		AcceptedBy:       d.AcceptedBy,
		ExtraData:        d.ExtraData,
	}
}

func WaveDemandAssignmentToDomain(p *WaveDemandAssignment) *domain.WaveDemandAssignment {
	return &domain.WaveDemandAssignment{
		ID:               p.ID,
		WaveID:           p.WaveID,
		DemandDocumentID: p.DemandDocumentID,
		AcceptedAt:       p.AcceptedAt,
		AcceptedBy:       p.AcceptedBy,
		ExtraData:        p.ExtraData,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// ---- Shipment ----

func ShipmentFromDomain(d *domain.Shipment) *Shipment {
	return &Shipment{
		SupplierOrderID:      d.SupplierOrderID,
		SupplierPlatform:     d.SupplierPlatform,
		ShipmentNo:           d.ShipmentNo,
		ExternalShipmentNo:   d.ExternalShipmentNo,
		CarrierCode:          d.CarrierCode,
		CarrierName:          d.CarrierName,
		TrackingNo:           d.TrackingNo,
		Status:               ShipmentStatus(d.Status),
		ShippedAt:            d.ShippedAt,
		BasisHistoryNodeID:   d.BasisHistoryNodeID,
		BasisProjectionHash:  d.BasisProjectionHash,
		BasisPayloadSnapshot: d.BasisPayloadSnapshot,
		ExtraData:            d.ExtraData,
	}
}

func ShipmentToDomain(p *Shipment) *domain.Shipment {
	return &domain.Shipment{
		ID:                   p.ID,
		SupplierOrderID:      p.SupplierOrderID,
		SupplierPlatform:     p.SupplierPlatform,
		ShipmentNo:           p.ShipmentNo,
		ExternalShipmentNo:   p.ExternalShipmentNo,
		CarrierCode:          p.CarrierCode,
		CarrierName:          p.CarrierName,
		TrackingNo:           p.TrackingNo,
		Status:               string(p.Status),
		ShippedAt:            p.ShippedAt,
		BasisHistoryNodeID:   p.BasisHistoryNodeID,
		BasisProjectionHash:  p.BasisProjectionHash,
		BasisPayloadSnapshot: p.BasisPayloadSnapshot,
		ExtraData:            p.ExtraData,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ---- ShipmentLine ----

func ShipmentLineFromDomain(d *domain.ShipmentLine) *ShipmentLine {
	return &ShipmentLine{
		ShipmentID:          d.ShipmentID,
		SupplierOrderLineID: d.SupplierOrderLineID,
		FulfillmentLineID:   d.FulfillmentLineID,
		Quantity:            d.Quantity,
	}
}

func ShipmentLineToDomain(p *ShipmentLine) *domain.ShipmentLine {
	return &domain.ShipmentLine{
		ID:                  p.ID,
		ShipmentID:          p.ShipmentID,
		SupplierOrderLineID: p.SupplierOrderLineID,
		FulfillmentLineID:   p.FulfillmentLineID,
		Quantity:            p.Quantity,
		CreatedAt:           p.CreatedAt,
	}
}

// ---- ChannelSyncJob ----

func ChannelSyncJobFromDomain(d *domain.ChannelSyncJob) *ChannelSyncJob {
	return &ChannelSyncJob{
		WaveID:               d.WaveID,
		IntegrationProfileID: d.IntegrationProfileID,
		Direction:            ChannelSyncDirection(d.Direction),
		Status:               ChannelSyncJobStatus(d.Status),
		BasisHistoryNodeID:   d.BasisHistoryNodeID,
		BasisProjectionHash:  d.BasisProjectionHash,
		BasisPayloadSnapshot: d.BasisPayloadSnapshot,
		RequestPayload:       d.RequestPayload,
		ResponsePayload:      d.ResponsePayload,
		ErrorMessage:         d.ErrorMessage,
		StartedAt:            d.StartedAt,
		FinishedAt:           d.FinishedAt,
	}
}

func ChannelSyncJobToDomain(p *ChannelSyncJob) *domain.ChannelSyncJob {
	return &domain.ChannelSyncJob{
		ID:                   p.ID,
		WaveID:               p.WaveID,
		IntegrationProfileID: p.IntegrationProfileID,
		Direction:            string(p.Direction),
		Status:               string(p.Status),
		BasisHistoryNodeID:   p.BasisHistoryNodeID,
		BasisProjectionHash:  p.BasisProjectionHash,
		BasisPayloadSnapshot: p.BasisPayloadSnapshot,
		RequestPayload:       p.RequestPayload,
		ResponsePayload:      p.ResponsePayload,
		ErrorMessage:         p.ErrorMessage,
		StartedAt:            p.StartedAt,
		FinishedAt:           p.FinishedAt,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ---- ChannelSyncItem ----

func ChannelSyncItemFromDomain(d *domain.ChannelSyncItem) *ChannelSyncItem {
	return &ChannelSyncItem{
		ChannelSyncJobID:   d.ChannelSyncJobID,
		FulfillmentLineID:  d.FulfillmentLineID,
		ShipmentID:         d.ShipmentID,
		ExternalDocumentNo: d.ExternalDocumentNo,
		ExternalLineNo:     d.ExternalLineNo,
		TrackingNo:         d.TrackingNo,
		CarrierCode:        d.CarrierCode,
		Status:             ChannelSyncItemStatus(d.Status),
		ErrorMessage:       d.ErrorMessage,
	}
}

func ChannelSyncItemToDomain(p *ChannelSyncItem) *domain.ChannelSyncItem {
	return &domain.ChannelSyncItem{
		ID:                 p.ID,
		ChannelSyncJobID:   p.ChannelSyncJobID,
		FulfillmentLineID:  p.FulfillmentLineID,
		ShipmentID:         p.ShipmentID,
		ExternalDocumentNo: p.ExternalDocumentNo,
		ExternalLineNo:     p.ExternalLineNo,
		TrackingNo:         p.TrackingNo,
		CarrierCode:        p.CarrierCode,
		Status:             string(p.Status),
		ErrorMessage:       p.ErrorMessage,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

// ---- IntegrationProfile ----

func IntegrationProfileFromDomain(d *domain.IntegrationProfile) *IntegrationProfile {
	return &IntegrationProfile{
		ProfileKey:                     d.ProfileKey,
		SourceChannel:                  d.SourceChannel,
		SourceSurface:                  d.SourceSurface,
		DemandKind:                     ProfileDemandKind(d.DemandKind),
		InitialAllocationStrategy:      InitialAllocationStrategy(d.InitialAllocationStrategy),
		IdentityStrategy:               IdentityStrategy(d.IdentityStrategy),
		EntitlementAuthorityMode:       EntitlementAuthorityMode(d.EntitlementAuthorityMode),
		RecipientInputMode:             RecipientInputMode(d.RecipientInputMode),
		ReferenceStrategy:              ReferenceStrategy(d.ReferenceStrategy),
		TrackingSyncMode:               TrackingSyncMode(d.TrackingSyncMode),
		ClosurePolicy:                  ClosurePolicy(d.ClosurePolicy),
		SupportsPartialShipment:        d.SupportsPartialShipment,
		SupportsAPIImport:              d.SupportsAPIImport,
		SupportsAPIExport:              d.SupportsAPIExport,
		RequiresCarrierMapping:         d.RequiresCarrierMapping,
		RequiresExternalOrderNo:        d.RequiresExternalOrderNo,
		AllowsManualClosure:            d.AllowsManualClosure,
		SupportsExportSupplierOrder:    d.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   d.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: d.SupportsImportSupplierShipment,
		ConnectorKey:                   d.ConnectorKey,
		FactorySupplierPlatform:        d.FactorySupplierPlatform,
		SupportedLocales:               d.SupportedLocales,
		DefaultLocale:                  d.DefaultLocale,
		ExtraData:                      d.ExtraData,
	}
}

func IntegrationProfileToDomain(p *IntegrationProfile) *domain.IntegrationProfile {
	return &domain.IntegrationProfile{
		ID:                             p.ID,
		ProfileKey:                     p.ProfileKey,
		SourceChannel:                  p.SourceChannel,
		SourceSurface:                  p.SourceSurface,
		DemandKind:                     string(p.DemandKind),
		InitialAllocationStrategy:      string(p.InitialAllocationStrategy),
		IdentityStrategy:               string(p.IdentityStrategy),
		EntitlementAuthorityMode:       string(p.EntitlementAuthorityMode),
		RecipientInputMode:             string(p.RecipientInputMode),
		ReferenceStrategy:              string(p.ReferenceStrategy),
		TrackingSyncMode:               string(p.TrackingSyncMode),
		ClosurePolicy:                  string(p.ClosurePolicy),
		SupportsPartialShipment:        p.SupportsPartialShipment,
		SupportsAPIImport:              p.SupportsAPIImport,
		SupportsAPIExport:              p.SupportsAPIExport,
		RequiresCarrierMapping:         p.RequiresCarrierMapping,
		RequiresExternalOrderNo:        p.RequiresExternalOrderNo,
		AllowsManualClosure:            p.AllowsManualClosure,
		SupportsExportSupplierOrder:    p.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   p.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: p.SupportsImportSupplierShipment,
		ConnectorKey:                   p.ConnectorKey,
		FactorySupplierPlatform:        p.FactorySupplierPlatform,
		SupportedLocales:               p.SupportedLocales,
		DefaultLocale:                  p.DefaultLocale,
		ExtraData:                      p.ExtraData,
		CreatedAt:                      p.CreatedAt,
		UpdatedAt:                      p.UpdatedAt,
	}
}

// ---- ChannelClosureDecisionRecord ----

func ChannelClosureDecisionRecordFromDomain(d *domain.ChannelClosureDecisionRecord) *ChannelClosureDecisionRecord {
	return &ChannelClosureDecisionRecord{
		WaveID:               d.WaveID,
		IntegrationProfileID: d.IntegrationProfileID,
		FulfillmentLineID:    d.FulfillmentLineID,
		DecisionKind:         ChannelClosureDecisionKind(d.DecisionKind),
		ReasonCode:           d.ReasonCode,
		Note:                 d.Note,
		EvidenceRef:          d.EvidenceRef,
		OperatorID:           d.OperatorID,
	}
}

func ChannelClosureDecisionRecordToDomain(p *ChannelClosureDecisionRecord) *domain.ChannelClosureDecisionRecord {
	return &domain.ChannelClosureDecisionRecord{
		ID:                   p.ID,
		WaveID:               p.WaveID,
		IntegrationProfileID: p.IntegrationProfileID,
		FulfillmentLineID:    p.FulfillmentLineID,
		DecisionKind:         string(p.DecisionKind),
		ReasonCode:           p.ReasonCode,
		Note:                 p.Note,
		EvidenceRef:          p.EvidenceRef,
		OperatorID:           p.OperatorID,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ---- FulfillmentAdjustment ----

func FulfillmentAdjustmentToDomain(p *FulfillmentAdjustment) *domain.FulfillmentAdjustment {
	return &domain.FulfillmentAdjustment{
		ID:                        p.ID,
		WaveID:                    p.WaveID,
		TargetKind:                p.TargetKind,
		FulfillmentLineID:         p.FulfillmentLineID,
		WaveParticipantSnapshotID: p.WaveParticipantSnapshotID,
		AdjustmentKind:            p.AdjustmentKind,
		QuantityDelta:             p.QuantityDelta,
		FromProductID:             p.FromProductID,
		ToProductID:               p.ToProductID,
		ReasonCode:                p.ReasonCode,
		OperatorID:                p.OperatorID,
		Note:                      p.Note,
		EvidenceRef:               p.EvidenceRef,
		CreatedAt:                 p.CreatedAt,
		UpdatedAt:                 p.UpdatedAt,
	}
}

func FulfillmentAdjustmentFromDomain(d *domain.FulfillmentAdjustment) *FulfillmentAdjustment {
	return &FulfillmentAdjustment{
		Model:                     gorm.Model{ID: d.ID},
		WaveID:                    d.WaveID,
		TargetKind:                d.TargetKind,
		FulfillmentLineID:         d.FulfillmentLineID,
		WaveParticipantSnapshotID: d.WaveParticipantSnapshotID,
		AdjustmentKind:            d.AdjustmentKind,
		QuantityDelta:             d.QuantityDelta,
		FromProductID:             d.FromProductID,
		ToProductID:               d.ToProductID,
		ReasonCode:                d.ReasonCode,
		OperatorID:                d.OperatorID,
		Note:                      d.Note,
		EvidenceRef:               d.EvidenceRef,
	}
}

// ---- DocumentTemplate ----

func DocumentTemplateToDomain(p *DocumentTemplate) *domain.DocumentTemplate {
	return &domain.DocumentTemplate{
		ID:           p.ID,
		TemplateKey:  p.TemplateKey,
		DocumentType: p.DocumentType,
		Format:       p.Format,
		MappingRules: p.MappingRules,
		ExtraData:    p.ExtraData,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func DocumentTemplateFromDomain(d *domain.DocumentTemplate) *DocumentTemplate {
	return &DocumentTemplate{
		Model:        gorm.Model{ID: d.ID},
		TemplateKey:  d.TemplateKey,
		DocumentType: d.DocumentType,
		Format:       d.Format,
		MappingRules: d.MappingRules,
		ExtraData:    d.ExtraData,
	}
}

// ---- IntegrationProfileTemplateBinding ----

func ProfileTemplateBindingToDomain(p *IntegrationProfileTemplateBinding) *domain.IntegrationProfileTemplateBinding {
	return &domain.IntegrationProfileTemplateBinding{
		ID:                   p.ID,
		IntegrationProfileID: p.IntegrationProfileID,
		DocumentType:         p.DocumentType,
		TemplateID:           p.TemplateID,
		IsDefault:            p.IsDefault,
		CreatedAt:            p.CreatedAt,
	}
}

func ProfileTemplateBindingFromDomain(d *domain.IntegrationProfileTemplateBinding) *IntegrationProfileTemplateBinding {
	return &IntegrationProfileTemplateBinding{
		Model:                gorm.Model{ID: d.ID},
		IntegrationProfileID: d.IntegrationProfileID,
		DocumentType:         d.DocumentType,
		TemplateID:           d.TemplateID,
		IsDefault:            d.IsDefault,
	}
}

// ---- HistoryScope ----

func HistoryScopeToDomain(p *HistoryScope) *domain.HistoryScope {
	return &domain.HistoryScope{
		ID:                p.ID,
		ScopeType:         p.ScopeType,
		ScopeKey:          p.ScopeKey,
		CurrentHeadNodeID: p.CurrentHeadNodeID,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func HistoryScopeFromDomain(d *domain.HistoryScope) *HistoryScope {
	return &HistoryScope{
		Model:             gorm.Model{ID: d.ID},
		ScopeType:         d.ScopeType,
		ScopeKey:          d.ScopeKey,
		CurrentHeadNodeID: d.CurrentHeadNodeID,
	}
}

// ---- HistoryNode ----

func HistoryNodeToDomain(p *HistoryNode) *domain.HistoryNode {
	return &domain.HistoryNode{
		ID:                   p.ID,
		HistoryScopeID:       p.HistoryScopeID,
		ParentNodeID:         p.ParentNodeID,
		PreferredRedoChildID: p.PreferredRedoChildID,
		CommandKind:          p.CommandKind,
		CommandSummary:       p.CommandSummary,
		PatchPayload:         p.PatchPayload,
		InversePatchPayload:  p.InversePatchPayload,
		CheckpointHint:       p.CheckpointHint,
		ProjectionHash:       p.ProjectionHash,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            p.CreatedAt,
	}
}

func HistoryNodeFromDomain(d *domain.HistoryNode) *HistoryNode {
	return &HistoryNode{
		Model:                gorm.Model{ID: d.ID},
		HistoryScopeID:       d.HistoryScopeID,
		ParentNodeID:         d.ParentNodeID,
		PreferredRedoChildID: d.PreferredRedoChildID,
		CommandKind:          d.CommandKind,
		CommandSummary:       d.CommandSummary,
		PatchPayload:         d.PatchPayload,
		InversePatchPayload:  d.InversePatchPayload,
		CheckpointHint:       d.CheckpointHint,
		ProjectionHash:       d.ProjectionHash,
		CreatedBy:            d.CreatedBy,
	}
}

// ---- HistoryCheckpoint ----

func HistoryCheckpointToDomain(p *HistoryCheckpoint) *domain.HistoryCheckpoint {
	return &domain.HistoryCheckpoint{
		ID:              p.ID,
		HistoryScopeID:  p.HistoryScopeID,
		HistoryNodeID:   p.HistoryNodeID,
		SnapshotPayload: p.SnapshotPayload,
		SchemaVersion:   p.SchemaVersion,
		CreatedAt:       p.CreatedAt,
	}
}

func HistoryCheckpointFromDomain(d *domain.HistoryCheckpoint) *HistoryCheckpoint {
	return &HistoryCheckpoint{
		Model:           gorm.Model{ID: d.ID},
		HistoryScopeID:  d.HistoryScopeID,
		HistoryNodeID:   d.HistoryNodeID,
		SnapshotPayload: d.SnapshotPayload,
		SchemaVersion:   d.SchemaVersion,
	}
}

// ---- HistoryPin ----

func HistoryPinToDomain(p *HistoryPin) *domain.HistoryPin {
	return &domain.HistoryPin{
		ID:            p.ID,
		HistoryNodeID: p.HistoryNodeID,
		PinKind:       p.PinKind,
		RefType:       p.RefType,
		RefID:         p.RefID,
		CreatedAt:     p.CreatedAt,
	}
}

func HistoryPinFromDomain(d *domain.HistoryPin) *HistoryPin {
	return &HistoryPin{
		Model:         gorm.Model{ID: d.ID},
		HistoryNodeID: d.HistoryNodeID,
		PinKind:       d.PinKind,
		RefType:       d.RefType,
		RefID:         d.RefID,
	}
}

// ---- ProductMaster ----

func ProductMasterFromDomain(d *domain.ProductMaster) *ProductMaster {
	p := &ProductMaster{
		SupplierPlatform:   d.SupplierPlatform,
		FactorySKU:         d.FactorySKU,
		SupplierProductRef: d.SupplierProductRef,
		Name:               d.Name,
		ProductKind:        string(d.ProductKind),
		Archived:           d.Archived,
		CoverImagePath:     d.CoverImagePath,
		DetailImagePaths:   d.DetailImagePaths,
		ExtraData:          d.ExtraData,
	}
	if d.ID != 0 {
		p.ID = d.ID
	}
	return p
}

func ProductMasterToDomain(p *ProductMaster) *domain.ProductMaster {
	return &domain.ProductMaster{
		ID:                 p.ID,
		SupplierPlatform:   p.SupplierPlatform,
		FactorySKU:         p.FactorySKU,
		SupplierProductRef: p.SupplierProductRef,
		Name:               p.Name,
		ProductKind:        domain.ProductKind(p.ProductKind),
		Archived:           p.Archived,
		CoverImagePath:     p.CoverImagePath,
		DetailImagePaths:   p.DetailImagePaths,
		ExtraData:          p.ExtraData,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

// ---- Product ----

func ProductFromDomain(d *domain.Product) *Product {
	p := &Product{
		WaveID:           d.WaveID,
		ProductMasterID:  d.ProductMasterID,
		SupplierPlatform: d.SupplierPlatform,
		FactorySKU:       d.FactorySKU,
		Name:             d.Name,
		ExtraData:        d.ExtraData,
	}
	if d.ID != 0 {
		p.ID = d.ID
	}
	return p
}

func ProductToDomain(p *Product) *domain.Product {
	return &domain.Product{
		ID:               p.ID,
		WaveID:           p.WaveID,
		ProductMasterID:  p.ProductMasterID,
		SupplierPlatform: p.SupplierPlatform,
		FactorySKU:       p.FactorySKU,
		Name:             p.Name,
		ExtraData:        p.ExtraData,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// ---- CarrierMapping ----

func CarrierMappingFromDomain(d *domain.CarrierMapping) *CarrierMapping {
	p := &CarrierMapping{
		IntegrationProfileID: d.IntegrationProfileID,
		InternalCarrierCode:  d.InternalCarrierCode,
		ExternalCarrierCode:  d.ExternalCarrierCode,
		ExternalCarrierName:  d.ExternalCarrierName,
		Aliases:              d.Aliases,
		IsDefault:            d.IsDefault,
	}
	if d.ID != 0 {
		p.ID = d.ID
	}
	return p
}

func CarrierMappingToDomain(p *CarrierMapping) *domain.CarrierMapping {
	return &domain.CarrierMapping{
		ID:                   p.ID,
		IntegrationProfileID: p.IntegrationProfileID,
		InternalCarrierCode:  p.InternalCarrierCode,
		ExternalCarrierCode:  p.ExternalCarrierCode,
		ExternalCarrierName:  p.ExternalCarrierName,
		Aliases:              p.Aliases,
		IsDefault:            p.IsDefault,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}
