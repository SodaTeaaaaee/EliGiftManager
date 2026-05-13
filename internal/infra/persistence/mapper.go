package persistence

import (
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- helper functions ----

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, _ := time.Parse(time.RFC3339, s)
	return &t
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ---- CustomerProfile ----

func ToPersistenceCustomerProfile(d *domain.CustomerProfile) *CustomerProfile {
	return &CustomerProfile{
		DisplayName: d.DisplayName,
		ProfileType: ProfileType(d.ProfileType),
		ExtraData:   d.ExtraData,
	}
}

func FromPersistenceCustomerProfile(p *CustomerProfile) *domain.CustomerProfile {
	return &domain.CustomerProfile{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		ProfileType: string(p.ProfileType),
		ExtraData:   p.ExtraData,
		CreatedAt:   formatTime(p.CreatedAt),
		UpdatedAt:   formatTime(p.UpdatedAt),
	}
}

// ---- CustomerIdentity ----

func ToPersistenceCustomerIdentity(d *domain.CustomerIdentity) *CustomerIdentity {
	return &CustomerIdentity{
		CustomerProfileID: d.CustomerProfileID,
		IdentityPlatform:  d.IdentityPlatform,
		IdentityValue:     d.IdentityValue,
		IdentityType:      IdentityType(d.IdentityType),
		IsPrimary:         d.IsPrimary,
		ExtraData:         d.ExtraData,
	}
}

func FromPersistenceCustomerIdentity(p *CustomerIdentity) *domain.CustomerIdentity {
	return &domain.CustomerIdentity{
		ID:                p.ID,
		CustomerProfileID: p.CustomerProfileID,
		IdentityPlatform:  p.IdentityPlatform,
		IdentityValue:     p.IdentityValue,
		IdentityType:      string(p.IdentityType),
		IsPrimary:         p.IsPrimary,
		ExtraData:         p.ExtraData,
		CreatedAt:         formatTime(p.CreatedAt),
		UpdatedAt:         formatTime(p.UpdatedAt),
	}
}

// ---- DemandDocument ----

func ToPersistenceDemandDocument(d *domain.DemandDocument) *DemandDocument {
	return &DemandDocument{
		Kind:                 DemandKind(d.Kind),
		CaptureMode:          CaptureMode(d.CaptureMode),
		SourceChannel:        d.SourceChannel,
		SourceSurface:        d.SourceSurface,
		IntegrationProfileID: d.IntegrationProfileID,
		SourceDocumentNo:     d.SourceDocumentNo,
		SourceCustomerRef:    d.SourceCustomerRef,
		CustomerProfileID:    d.CustomerProfileID,
		SourceCreatedAt:      parseTimePtr(d.SourceCreatedAt),
		SourcePaidAt:         parseTimePtr(d.SourcePaidAt),
		Currency:             d.Currency,
		AuthoritySnapshotAt:  parseTimePtr(d.AuthoritySnapshotAt),
		RawPayload:           d.RawPayload,
		ExtraData:            d.ExtraData,
	}
}

func FromPersistenceDemandDocument(p *DemandDocument) *domain.DemandDocument {
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
		SourceCreatedAt:      formatTimePtr(p.SourceCreatedAt),
		SourcePaidAt:         formatTimePtr(p.SourcePaidAt),
		Currency:             p.Currency,
		AuthoritySnapshotAt:  formatTimePtr(p.AuthoritySnapshotAt),
		RawPayload:           p.RawPayload,
		ExtraData:            p.ExtraData,
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
	}
}

// ---- DemandLine ----

func ToPersistenceDemandLine(d *domain.DemandLine) *DemandLine {
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

func FromPersistenceDemandLine(p *DemandLine) *domain.DemandLine {
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
		CreatedAt:             formatTime(p.CreatedAt),
		UpdatedAt:             formatTime(p.UpdatedAt),
	}
}

// ---- Wave ----

func ToPersistenceWave(d *domain.Wave) *Wave {
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

func FromPersistenceWave(p *Wave) *domain.Wave {
	return &domain.Wave{
		ID:               p.ID,
		WaveNo:           p.WaveNo,
		Name:             p.Name,
		WaveType:         string(p.WaveType),
		LifecycleStage:   p.LifecycleStage,
		ProgressSnapshot: p.ProgressSnapshot,
		Notes:            p.Notes,
		LevelTags:        p.LevelTags,
		CreatedAt:        formatTime(p.CreatedAt),
		UpdatedAt:        formatTime(p.UpdatedAt),
	}
}

// ---- WaveParticipantSnapshot ----

func ToPersistenceWaveParticipantSnapshot(d *domain.WaveParticipantSnapshot) *WaveParticipantSnapshot {
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

func FromPersistenceWaveParticipantSnapshot(p *WaveParticipantSnapshot) *domain.WaveParticipantSnapshot {
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
		CreatedAt:          formatTime(p.CreatedAt),
	}
}

// ---- FulfillmentLine ----

func ToPersistenceFulfillmentLine(d *domain.FulfillmentLine) *FulfillmentLine {
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

func FromPersistenceFulfillmentLine(p *FulfillmentLine) *domain.FulfillmentLine {
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
		CreatedAt:                 formatTime(p.CreatedAt),
		UpdatedAt:                 formatTime(p.UpdatedAt),
	}
}

// ---- AllocationPolicyRule ----

func ToPersistenceAllocationPolicyRule(d *domain.AllocationPolicyRule) *AllocationPolicyRule {
	return &AllocationPolicyRule{
		WaveID:               d.WaveID,
		ProductID:            d.ProductID,
		SelectorPayload:      d.SelectorPayload,
		ProductTargetRef:     d.ProductTargetRef,
		ContributionQuantity: d.ContributionQuantity,
		RuleKind:             d.RuleKind,
		Priority:             d.Priority,
		Active:               d.Active,
	}
}

func FromPersistenceAllocationPolicyRule(p *AllocationPolicyRule) *domain.AllocationPolicyRule {
	return &domain.AllocationPolicyRule{
		ID:                   p.ID,
		WaveID:               p.WaveID,
		ProductID:            p.ProductID,
		SelectorPayload:      p.SelectorPayload,
		ProductTargetRef:     p.ProductTargetRef,
		ContributionQuantity: p.ContributionQuantity,
		RuleKind:             p.RuleKind,
		Priority:             p.Priority,
		Active:               p.Active,
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
	}
}

// ---- SupplierOrder ----

func ToPersistenceSupplierOrder(d *domain.SupplierOrder) *SupplierOrder {
	return &SupplierOrder{
		WaveID:               d.WaveID,
		SupplierPlatform:     d.SupplierPlatform,
		TemplateID:           d.TemplateID,
		BatchNo:              d.BatchNo,
		ExternalOrderNo:      d.ExternalOrderNo,
		SubmissionMode:       SubmissionMode(d.SubmissionMode),
		SubmittedAt:          parseTimePtr(d.SubmittedAt),
		Status:               SupplierOrderStatus(d.Status),
		RequestPayload:       d.RequestPayload,
		ResponsePayload:      d.ResponsePayload,
		BasisHistoryNodeID:   d.BasisHistoryNodeID,
		BasisProjectionHash:  d.BasisProjectionHash,
		BasisPayloadSnapshot: d.BasisPayloadSnapshot,
		ExtraData:            d.ExtraData,
	}
}

func FromPersistenceSupplierOrder(p *SupplierOrder) *domain.SupplierOrder {
	return &domain.SupplierOrder{
		ID:                   p.ID,
		WaveID:               p.WaveID,
		SupplierPlatform:     p.SupplierPlatform,
		TemplateID:           p.TemplateID,
		BatchNo:              p.BatchNo,
		ExternalOrderNo:      p.ExternalOrderNo,
		SubmissionMode:       string(p.SubmissionMode),
		SubmittedAt:          formatTimePtr(p.SubmittedAt),
		Status:               string(p.Status),
		RequestPayload:       p.RequestPayload,
		ResponsePayload:      p.ResponsePayload,
		BasisHistoryNodeID:   p.BasisHistoryNodeID,
		BasisProjectionHash:  p.BasisProjectionHash,
		BasisPayloadSnapshot: p.BasisPayloadSnapshot,
		ExtraData:            p.ExtraData,
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
	}
}

// ---- SupplierOrderLine ----

func ToPersistenceSupplierOrderLine(d *domain.SupplierOrderLine) *SupplierOrderLine {
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

func FromPersistenceSupplierOrderLine(p *SupplierOrderLine) *domain.SupplierOrderLine {
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
		CreatedAt:         formatTime(p.CreatedAt),
		UpdatedAt:         formatTime(p.UpdatedAt),
	}
}

// ---- WaveDemandAssignment ----

func ToPersistenceWaveDemandAssignment(d *domain.WaveDemandAssignment) *WaveDemandAssignment {
	return &WaveDemandAssignment{
		WaveID:           d.WaveID,
		DemandDocumentID: d.DemandDocumentID,
		AcceptedAt:       parseTimePtr(d.AcceptedAt),
		AcceptedBy:       d.AcceptedBy,
		ExtraData:        d.ExtraData,
	}
}

func FromPersistenceWaveDemandAssignment(p *WaveDemandAssignment) *domain.WaveDemandAssignment {
	return &domain.WaveDemandAssignment{
		ID:               p.ID,
		WaveID:           p.WaveID,
		DemandDocumentID: p.DemandDocumentID,
		AcceptedAt:       formatTimePtr(p.AcceptedAt),
		AcceptedBy:       p.AcceptedBy,
		ExtraData:        p.ExtraData,
		CreatedAt:        formatTime(p.CreatedAt),
		UpdatedAt:        formatTime(p.UpdatedAt),
	}
}

// ---- Shipment ----

func ToPersistenceShipment(d *domain.Shipment) *Shipment {
	return &Shipment{
		SupplierOrderID:      d.SupplierOrderID,
		SupplierPlatform:     d.SupplierPlatform,
		ShipmentNo:           d.ShipmentNo,
		ExternalShipmentNo:   d.ExternalShipmentNo,
		CarrierCode:          d.CarrierCode,
		CarrierName:          d.CarrierName,
		TrackingNo:           d.TrackingNo,
		Status:               ShipmentStatus(d.Status),
		ShippedAt:            parseTimePtr(d.ShippedAt),
		BasisHistoryNodeID:   d.BasisHistoryNodeID,
		BasisProjectionHash:  d.BasisProjectionHash,
		BasisPayloadSnapshot: d.BasisPayloadSnapshot,
		ExtraData:            d.ExtraData,
	}
}

func FromPersistenceShipment(p *Shipment) *domain.Shipment {
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
		ShippedAt:            formatTimePtr(p.ShippedAt),
		BasisHistoryNodeID:   p.BasisHistoryNodeID,
		BasisProjectionHash:  p.BasisProjectionHash,
		BasisPayloadSnapshot: p.BasisPayloadSnapshot,
		ExtraData:            p.ExtraData,
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
	}
}

// ---- ShipmentLine ----

func ToPersistenceShipmentLine(d *domain.ShipmentLine) *ShipmentLine {
	return &ShipmentLine{
		ShipmentID:          d.ShipmentID,
		SupplierOrderLineID: d.SupplierOrderLineID,
		FulfillmentLineID:   d.FulfillmentLineID,
		Quantity:            d.Quantity,
	}
}

func FromPersistenceShipmentLine(p *ShipmentLine) *domain.ShipmentLine {
	return &domain.ShipmentLine{
		ID:                  p.ID,
		ShipmentID:          p.ShipmentID,
		SupplierOrderLineID: p.SupplierOrderLineID,
		FulfillmentLineID:   p.FulfillmentLineID,
		Quantity:            p.Quantity,
		CreatedAt:           formatTime(p.CreatedAt),
	}
}
