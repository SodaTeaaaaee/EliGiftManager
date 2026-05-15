package persistence

import (
	"encoding/json"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
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
	selectorJSON, _ := json.Marshal(d.SelectorPayload)
	return &AllocationPolicyRule{
		WaveID:               d.WaveID,
		ProductID:            d.ProductID,
		SelectorPayload:      string(selectorJSON),
		ProductTargetRef:     d.ProductTargetRef,
		ContributionQuantity: d.ContributionQuantity,
		RuleKind:             d.RuleKind,
		Priority:             d.Priority,
		Active:               d.Active,
	}
}

func FromPersistenceAllocationPolicyRule(p *AllocationPolicyRule) *domain.AllocationPolicyRule {
	var selector domain.SelectorPayload
	if p.SelectorPayload != "" {
		_ = json.Unmarshal([]byte(p.SelectorPayload), &selector)
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

// ---- ChannelSyncJob ----

func ToPersistenceChannelSyncJob(d *domain.ChannelSyncJob) *ChannelSyncJob {
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
		StartedAt:            parseTimePtr(d.StartedAt),
		FinishedAt:           parseTimePtr(d.FinishedAt),
	}
}

func FromPersistenceChannelSyncJob(p *ChannelSyncJob) *domain.ChannelSyncJob {
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
		StartedAt:            formatTimePtr(p.StartedAt),
		FinishedAt:           formatTimePtr(p.FinishedAt),
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
	}
}

// ---- ChannelSyncItem ----

func ToPersistenceChannelSyncItem(d *domain.ChannelSyncItem) *ChannelSyncItem {
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

func FromPersistenceChannelSyncItem(p *ChannelSyncItem) *domain.ChannelSyncItem {
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
		CreatedAt:          formatTime(p.CreatedAt),
		UpdatedAt:          formatTime(p.UpdatedAt),
	}
}

// ---- IntegrationProfile ----

func ToPersistenceIntegrationProfile(d *domain.IntegrationProfile) *IntegrationProfile {
	return &IntegrationProfile{
		ProfileKey:                d.ProfileKey,
		SourceChannel:             d.SourceChannel,
		SourceSurface:             d.SourceSurface,
		DemandKind:                ProfileDemandKind(d.DemandKind),
		InitialAllocationStrategy: InitialAllocationStrategy(d.InitialAllocationStrategy),
		IdentityStrategy:          IdentityStrategy(d.IdentityStrategy),
		EntitlementAuthorityMode:  EntitlementAuthorityMode(d.EntitlementAuthorityMode),
		RecipientInputMode:        RecipientInputMode(d.RecipientInputMode),
		ReferenceStrategy:         ReferenceStrategy(d.ReferenceStrategy),
		TrackingSyncMode:          TrackingSyncMode(d.TrackingSyncMode),
		ClosurePolicy:             ClosurePolicy(d.ClosurePolicy),
		SupportsPartialShipment:   d.SupportsPartialShipment,
		SupportsAPIImport:         d.SupportsAPIImport,
		SupportsAPIExport:         d.SupportsAPIExport,
		RequiresCarrierMapping:    d.RequiresCarrierMapping,
		RequiresExternalOrderNo:   d.RequiresExternalOrderNo,
		AllowsManualClosure:       d.AllowsManualClosure,
		ConnectorKey:              d.ConnectorKey,
		SupportedLocales:          d.SupportedLocales,
		DefaultLocale:             d.DefaultLocale,
		ExtraData:                 d.ExtraData,
	}
}

func FromPersistenceIntegrationProfile(p *IntegrationProfile) *domain.IntegrationProfile {
	return &domain.IntegrationProfile{
		ID:                        p.ID,
		ProfileKey:                p.ProfileKey,
		SourceChannel:             p.SourceChannel,
		SourceSurface:             p.SourceSurface,
		DemandKind:                string(p.DemandKind),
		InitialAllocationStrategy: string(p.InitialAllocationStrategy),
		IdentityStrategy:          string(p.IdentityStrategy),
		EntitlementAuthorityMode:  string(p.EntitlementAuthorityMode),
		RecipientInputMode:        string(p.RecipientInputMode),
		ReferenceStrategy:         string(p.ReferenceStrategy),
		TrackingSyncMode:          string(p.TrackingSyncMode),
		ClosurePolicy:             string(p.ClosurePolicy),
		SupportsPartialShipment:   p.SupportsPartialShipment,
		SupportsAPIImport:         p.SupportsAPIImport,
		SupportsAPIExport:         p.SupportsAPIExport,
		RequiresCarrierMapping:    p.RequiresCarrierMapping,
		RequiresExternalOrderNo:   p.RequiresExternalOrderNo,
		AllowsManualClosure:       p.AllowsManualClosure,
		ConnectorKey:              p.ConnectorKey,
		SupportedLocales:          p.SupportedLocales,
		DefaultLocale:             p.DefaultLocale,
		ExtraData:                 p.ExtraData,
		CreatedAt:                 formatTime(p.CreatedAt),
		UpdatedAt:                 formatTime(p.UpdatedAt),
	}
}

// ---- ChannelClosureDecisionRecord ----

func ToPersistenceChannelClosureDecisionRecord(d *domain.ChannelClosureDecisionRecord) *ChannelClosureDecisionRecord {
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

func FromPersistenceChannelClosureDecisionRecord(p *ChannelClosureDecisionRecord) *domain.ChannelClosureDecisionRecord {
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
		CreatedAt:            formatTime(p.CreatedAt),
		UpdatedAt:            formatTime(p.UpdatedAt),
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
		ReasonCode:                p.ReasonCode,
		OperatorID:                p.OperatorID,
		Note:                      p.Note,
		EvidenceRef:               p.EvidenceRef,
		CreatedAt:                 p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                 p.UpdatedAt.Format(time.RFC3339),
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
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
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
		CreatedAt:            p.CreatedAt.Format(time.RFC3339),
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
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         p.UpdatedAt.Format(time.RFC3339),
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
		CreatedAt:            p.CreatedAt.Format(time.RFC3339),
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
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
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
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
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

func ToPersistenceProductMaster(d *domain.ProductMaster) *ProductMaster {
	p := &ProductMaster{
		SupplierPlatform:   d.SupplierPlatform,
		FactorySKU:         d.FactorySKU,
		SupplierProductRef: d.SupplierProductRef,
		Name:               d.Name,
		ProductKind:        string(d.ProductKind),
		Archived:           d.Archived,
		ExtraData:          d.ExtraData,
	}
	if d.ID != 0 {
		p.ID = d.ID
	}
	return p
}

func FromPersistenceProductMaster(p *ProductMaster) *domain.ProductMaster {
	return &domain.ProductMaster{
		ID:                 p.ID,
		SupplierPlatform:   p.SupplierPlatform,
		FactorySKU:         p.FactorySKU,
		SupplierProductRef: p.SupplierProductRef,
		Name:               p.Name,
		ProductKind:        domain.ProductKind(p.ProductKind),
		Archived:           p.Archived,
		ExtraData:          p.ExtraData,
		CreatedAt:          formatTime(p.CreatedAt),
		UpdatedAt:          formatTime(p.UpdatedAt),
	}
}

// ---- Product ----

func ToPersistenceProduct(d *domain.Product) *Product {
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

func FromPersistenceProduct(p *Product) *domain.Product {
	return &domain.Product{
		ID:               p.ID,
		WaveID:           p.WaveID,
		ProductMasterID:  p.ProductMasterID,
		SupplierPlatform: p.SupplierPlatform,
		FactorySKU:       p.FactorySKU,
		Name:             p.Name,
		ExtraData:        p.ExtraData,
		CreatedAt:        formatTime(p.CreatedAt),
		UpdatedAt:        formatTime(p.UpdatedAt),
	}
}
