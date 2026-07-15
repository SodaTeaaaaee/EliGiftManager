package domain

import (
	"context"
	"time"
)

// CustomerProfileRepository defines persistence operations for CustomerProfile and CustomerIdentity.
type CustomerProfileRepository interface {
	Create(ctx context.Context, profile *CustomerProfile) error
	Update(ctx context.Context, profile *CustomerProfile) error
	FindByID(ctx context.Context, id uint) (*CustomerProfile, error)
	List(ctx context.Context) ([]CustomerProfile, error)

	CreateIdentity(ctx context.Context, identity *CustomerIdentity) error
	ListIdentitiesByProfile(ctx context.Context, profileID uint) ([]CustomerIdentity, error)
	FindIdentityByPlatformAndValue(ctx context.Context, platform, value string) (*CustomerIdentity, error)
	UpdateIdentityProfileID(ctx context.Context, identityID uint, newProfileID uint) error
	BulkUpdateIdentityProfileID(ctx context.Context, identityIDs []uint, newProfileID uint) error
	SoftDelete(ctx context.Context, id uint) error
	DeleteIdentity(ctx context.Context, id uint) error
}

// CustomerMergeProfileRepository adds integrity operations used only by merge flows.
type CustomerMergeProfileRepository interface {
	CustomerProfileRepository
	ListIdentitiesByIDs(ctx context.Context, identityIDs []uint) ([]CustomerIdentity, error)
	IsSoftDeleted(ctx context.Context, id uint) (bool, error)
	RestoreSoftDeleted(ctx context.Context, id uint) error
}

// CustomerAddressRepository defines persistence operations for CustomerAddress.
type CustomerAddressRepository interface {
	Create(ctx context.Context, addr *CustomerAddress) error
	FindByID(ctx context.Context, id uint) (*CustomerAddress, error)
	ListByProfile(ctx context.Context, profileID uint) ([]CustomerAddress, error)
	Update(ctx context.Context, addr *CustomerAddress) error
	SoftDelete(ctx context.Context, id uint) error
	ClearDefaultByProfile(ctx context.Context, profileID uint) error
	BulkUpdateProfileID(ctx context.Context, oldProfileID, newProfileID uint) error
}

// CustomerMergeAddressRepository adds exact-row operations used by merge flows.
type CustomerMergeAddressRepository interface {
	CustomerAddressRepository
	ListByIDs(ctx context.Context, addressIDs []uint) ([]CustomerAddress, error)
	BulkUpdateProfileIDByIDs(ctx context.Context, addressIDs []uint, newProfileID uint) error
}

// DemandDocumentRepository defines persistence operations for DemandDocument and DemandLine.
type DemandDocumentRepository interface {
	Create(ctx context.Context, doc *DemandDocument) error
	FindByID(ctx context.Context, id uint) (*DemandDocument, error)
	List(ctx context.Context) ([]DemandDocument, error)
	ListUnassigned(ctx context.Context) ([]DemandDocument, error)
	CountByIntegrationProfileID(ctx context.Context, profileID uint) (int64, error)

	// UpdateBoundProfileSnapshot persists only the BoundProfileSnapshot field for the given document ID.
	// Used at wave assignment time and during explicit profile refresh.
	UpdateBoundProfileSnapshot(ctx context.Context, docID uint, snapshot string) error

	// BulkUpdateCustomerProfileID reassigns all demand documents from oldProfileID to newProfileID.
	// Returns the number of rows updated.
	BulkUpdateCustomerProfileID(ctx context.Context, oldProfileID, newProfileID uint) (int64, error)

	CreateLine(ctx context.Context, line *DemandLine) error
	FindLineByID(ctx context.Context, id uint) (*DemandLine, error)
	ListLinesByDocument(ctx context.Context, docID uint) ([]DemandLine, error)
	UpdateLine(ctx context.Context, line *DemandLine) error
	UpdateLineRoutingFields(ctx context.Context, lineID uint, routingDisposition string, recipientInputState string, routingReasonCode string) error
}

// CustomerMergeDemandRepository adds exact-row and assignment-aware operations
// used by merge flows.
type CustomerMergeDemandRepository interface {
	DemandDocumentRepository
	ListByIDs(ctx context.Context, documentIDs []uint) ([]DemandDocument, error)
	ListUnassignedByCustomerProfileID(ctx context.Context, profileID uint) ([]DemandDocument, error)
	BulkUpdateCustomerProfileIDByIDs(ctx context.Context, documentIDs []uint, newProfileID uint) error
}

// CustomerMergeRecordRepository persists reversible merge audit records.
type CustomerMergeRecordRepository interface {
	Create(ctx context.Context, record *CustomerMergeRecord) error
	FindByID(ctx context.Context, id uint) (*CustomerMergeRecord, error)
	ListActiveByTargetProfileID(ctx context.Context, targetProfileID uint) ([]CustomerMergeRecord, error)
	MarkUndone(ctx context.Context, id uint, undoneAt time.Time) error
}

// WaveRepository defines persistence operations for Wave and WaveParticipantSnapshot.
type WaveRepository interface {
	Create(ctx context.Context, wave *Wave) error
	FindByID(ctx context.Context, id uint) (*Wave, error)
	FindByWaveNo(ctx context.Context, waveNo string) (*Wave, error)
	List(ctx context.Context) ([]Wave, error)
	// ListPaginated returns a page of waves (ordered by id) plus the total count,
	// pushing LIMIT/OFFSET down to SQL instead of slicing in memory.
	ListPaginated(ctx context.Context, offset, limit int) ([]Wave, int64, error)
	UpdateLifecycle(ctx context.Context, waveID uint, stage string, progressSnapshot string) error

	AddParticipant(ctx context.Context, snap *WaveParticipantSnapshot) error
	ListParticipantsByWave(ctx context.Context, waveID uint) ([]WaveParticipantSnapshot, error)
	ListParticipantsByProfile(ctx context.Context, profileID uint) ([]WaveParticipantSnapshot, error)
	UpdateParticipantProfileID(ctx context.Context, oldProfileID, newProfileID uint) (int64, error)
	DeleteParticipantsByWave(ctx context.Context, waveID uint) error
	CountByDatePrefix(ctx context.Context, prefix string) (int, error)
}

// FulfillmentLineRepository defines persistence operations for FulfillmentLine.
type FulfillmentLineStateUpdate struct {
	ID               uint
	SupplierState    string
	ChannelSyncState string
}

type FulfillmentLineRepository interface {
	Create(ctx context.Context, line *FulfillmentLine) error
	FindByID(ctx context.Context, id uint) (*FulfillmentLine, error)
	ListByWave(ctx context.Context, waveID uint) ([]FulfillmentLine, error)
	Update(ctx context.Context, line *FulfillmentLine) error
	DeleteByWave(ctx context.Context, waveID uint) error
	DeleteByWaveAndGeneratedBy(ctx context.Context, waveID uint, generatedBy string) error
	ReplaceByWaveAndGeneratedBy(ctx context.Context, waveID uint, generatedBy string, newLines []FulfillmentLine) error
	BulkUpdateStates(ctx context.Context, updates []FulfillmentLineStateUpdate) error
	BulkUpdateCustomerProfileID(ctx context.Context, oldProfileID, newProfileID uint) (int64, error)
}

// SupplierOrderRepository defines persistence operations for SupplierOrder and SupplierOrderLine.
type SupplierOrderRepository interface {
	Create(ctx context.Context, order *SupplierOrder) error
	FindByID(ctx context.Context, id uint) (*SupplierOrder, error)
	List(ctx context.Context) ([]SupplierOrder, error)
	ListByWave(ctx context.Context, waveID uint) ([]SupplierOrder, error)
	DeleteDraftsByWave(ctx context.Context, waveID uint) error

	CreateLine(ctx context.Context, line *SupplierOrderLine) error
	ListLinesByOrder(ctx context.Context, orderID uint) ([]SupplierOrderLine, error)
	FindLineByID(ctx context.Context, id uint) (*SupplierOrderLine, error)
	DeleteLinesByOrder(ctx context.Context, orderID uint) error

	// AtomicCreateSupplierOrder creates order + lines + optional basis pin in one transaction.
	AtomicCreateSupplierOrder(ctx context.Context, order *SupplierOrder, lines []*SupplierOrderLine, pin *BasisPinParam) error

	Update(ctx context.Context, order *SupplierOrder) error
}

// AllocationPolicyRuleRepository defines persistence operations for AllocationPolicyRule.
type AllocationPolicyRuleRepository interface {
	Create(ctx context.Context, rule *AllocationPolicyRule) error
	FindByID(ctx context.Context, id uint) (*AllocationPolicyRule, error)
	ListByWave(ctx context.Context, waveID uint) ([]AllocationPolicyRule, error)
	Update(ctx context.Context, rule *AllocationPolicyRule) error
	Delete(ctx context.Context, id uint) error
	DeleteByWave(ctx context.Context, waveID uint) error
}

// WaveDemandAssignmentRepository defines persistence operations for wave-demand linkage.
type WaveDemandAssignmentRepository interface {
	Create(ctx context.Context, assignment *WaveDemandAssignment) error
	DeleteByWaveAndDocument(ctx context.Context, waveID uint, demandDocumentID uint) error
	DeleteByWave(ctx context.Context, waveID uint) error
	ListByWave(ctx context.Context, waveID uint) ([]WaveDemandAssignment, error)
	ListByDemandDocument(ctx context.Context, docID uint) ([]WaveDemandAssignment, error)
	ListDemandDocumentsByWave(ctx context.Context, waveID uint) ([]DemandDocument, error)
}

// ShipmentRepository defines persistence operations for Shipment and ShipmentLine.
type ShipmentRepository interface {
	Create(ctx context.Context, shipment *Shipment) error
	FindByID(ctx context.Context, id uint) (*Shipment, error)
	ListBySupplierOrder(ctx context.Context, supplierOrderID uint) ([]Shipment, error)
	ListByWave(ctx context.Context, waveID uint) ([]Shipment, error)

	CreateLine(ctx context.Context, line *ShipmentLine) error
	ListLinesByShipment(ctx context.Context, shipmentID uint) ([]ShipmentLine, error)

	// SumShippedQuantityBySOL returns the total quantity already shipped for a given
	// SupplierOrderLine across all existing shipments. Used for cumulative over-shipment
	// validation before persisting a new shipment.
	SumShippedQuantityBySOL(ctx context.Context, supplierOrderLineID uint) (int, error)

	// AtomicCreateShipment creates a shipment, its lines, and optional basis pin atomically.
	AtomicCreateShipment(ctx context.Context, shipment *Shipment, lines []*ShipmentLine, pin *BasisPinParam) error
}

// ChannelSyncRepository defines persistence operations for ChannelSyncJob and ChannelSyncItem.
type ChannelSyncRepository interface {
	CreateJob(ctx context.Context, job *ChannelSyncJob) error
	FindJobByID(ctx context.Context, id uint) (*ChannelSyncJob, error)
	ListJobsByWave(ctx context.Context, waveID uint) ([]ChannelSyncJob, error)
	SaveJob(ctx context.Context, job *ChannelSyncJob) error

	CreateItem(ctx context.Context, item *ChannelSyncItem) error
	SaveItem(ctx context.Context, item *ChannelSyncItem) error
	ListItemsByJob(ctx context.Context, jobID uint) ([]ChannelSyncItem, error)

	// AtomicCreateChannelSync creates a job, its items, and optional basis pin atomically.
	AtomicCreateChannelSync(ctx context.Context, job *ChannelSyncJob, items []*ChannelSyncItem, pin *BasisPinParam) error

	CountJobsByProfileID(ctx context.Context, profileID uint) (int64, error)
}

// ChannelClosureDecisionRepository defines persistence operations for channel closure decision records.
type ChannelClosureDecisionRepository interface {
	Create(ctx context.Context, record *ChannelClosureDecisionRecord) error
	AtomicCreate(ctx context.Context, records []*ChannelClosureDecisionRecord) error
	ListByFulfillmentLine(ctx context.Context, fulfillmentLineID uint) ([]ChannelClosureDecisionRecord, error)
	ListByWave(ctx context.Context, waveID uint) ([]ChannelClosureDecisionRecord, error)
	CountByProfileID(ctx context.Context, profileID uint) (int64, error)
}

// IntegrationProfileRepository defines persistence operations for IntegrationProfile.
type IntegrationProfileRepository interface {
	Create(ctx context.Context, profile *IntegrationProfile) error
	FindByID(ctx context.Context, id uint) (*IntegrationProfile, error)
	FindByProfileKey(ctx context.Context, profileKey string) (*IntegrationProfile, error)
	List(ctx context.Context) ([]IntegrationProfile, error)
	Update(ctx context.Context, profile *IntegrationProfile) error
	Delete(ctx context.Context, id uint) error
}

// FulfillmentAdjustmentRepository defines persistence operations for FulfillmentAdjustment.
type FulfillmentAdjustmentRepository interface {
	Create(ctx context.Context, adj *FulfillmentAdjustment) error
	FindByID(ctx context.Context, id uint) (*FulfillmentAdjustment, error)
	Update(ctx context.Context, adj *FulfillmentAdjustment) error
	Delete(ctx context.Context, id uint) error
	DeleteByWave(ctx context.Context, waveID uint) error
	ListByWave(ctx context.Context, waveID uint) ([]FulfillmentAdjustment, error)
	ListByFulfillmentLine(ctx context.Context, fulfillmentLineID uint) ([]FulfillmentAdjustment, error)
}

// DocumentTemplateRepository defines persistence operations for DocumentTemplate.
type DocumentTemplateRepository interface {
	Create(ctx context.Context, t *DocumentTemplate) error
	FindByID(ctx context.Context, id uint) (*DocumentTemplate, error)
	FindByKey(ctx context.Context, key string) (*DocumentTemplate, error)
	List(ctx context.Context) ([]DocumentTemplate, error)
	ListByDocumentType(ctx context.Context, docType string) ([]DocumentTemplate, error)
	Update(ctx context.Context, t *DocumentTemplate) error
	Delete(ctx context.Context, id uint) error
}

// ProfileTemplateBindingRepository defines persistence operations for IntegrationProfileTemplateBinding.
type ProfileTemplateBindingRepository interface {
	Create(ctx context.Context, b *IntegrationProfileTemplateBinding) error
	FindByID(ctx context.Context, id uint) (*IntegrationProfileTemplateBinding, error)
	ListByProfile(ctx context.Context, profileID uint) ([]IntegrationProfileTemplateBinding, error)
	ListByTemplateID(ctx context.Context, templateID uint) ([]IntegrationProfileTemplateBinding, error)
	FindDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) (*IntegrationProfileTemplateBinding, error)
	// ClearDefaultByProfileAndType sets is_default=false for all bindings of the
	// given (profileID, documentType) pair. Used by SetDefaultBinding to enforce
	// uniqueness before promoting a new default.
	ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error
	Update(ctx context.Context, b *IntegrationProfileTemplateBinding) error
	Delete(ctx context.Context, id uint) error
	CountByProfileID(ctx context.Context, profileID uint) (int64, error)
}

// HistoryScopeRepository defines persistence operations for HistoryScope.
type HistoryScopeRepository interface {
	Create(ctx context.Context, scope *HistoryScope) error
	FindByID(ctx context.Context, id uint) (*HistoryScope, error)
	FindByScopeTypeAndKey(ctx context.Context, scopeType string, scopeKey string) (*HistoryScope, error)
	UpdateHead(ctx context.Context, scopeID uint, headNodeID uint) error
	FindOrCreate(ctx context.Context, scopeType string, scopeKey string) (*HistoryScope, error)
}

// HistoryNodeRepository defines persistence operations for HistoryNode.
type HistoryNodeRepository interface {
	Create(ctx context.Context, node *HistoryNode) error
	FindByID(ctx context.Context, id uint) (*HistoryNode, error)
	UpdatePreferredRedoChild(ctx context.Context, nodeID uint, childID uint) error
	ListByScopeRecent(ctx context.Context, scopeID uint, limit int) ([]HistoryNode, error)
	ListByScope(ctx context.Context, scopeID uint) ([]HistoryNode, error)
	DeleteByID(ctx context.Context, nodeID uint) error
}

// HistoryCheckpointRepository defines persistence operations for HistoryCheckpoint.
type HistoryCheckpointRepository interface {
	Create(ctx context.Context, cp *HistoryCheckpoint) error
	FindByNodeID(ctx context.Context, nodeID uint) (*HistoryCheckpoint, error)
	DeleteByNodeID(ctx context.Context, nodeID uint) error
}

// HistoryPinRepository defines persistence operations for HistoryPin.
type HistoryPinRepository interface {
	Create(ctx context.Context, pin *HistoryPin) error
	ListByNodeID(ctx context.Context, nodeID uint) ([]HistoryPin, error)
	CountByNodeID(ctx context.Context, nodeID uint) (int64, error)
	ListPinnedNodeIDsByScope(ctx context.Context, scopeID uint) ([]uint, error)
}

// ProductMasterRepository defines persistence operations for ProductMaster.
type ProductMasterRepository interface {
	Create(ctx context.Context, master *ProductMaster) error
	FindByID(ctx context.Context, id uint) (*ProductMaster, error)
	List(ctx context.Context) ([]ProductMaster, error)
	FindByPlatformAndSKU(ctx context.Context, platform, sku string) (*ProductMaster, error)
	Update(ctx context.Context, master *ProductMaster) error
}

// ProductRepository defines persistence operations for Product.
type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	FindByID(ctx context.Context, id uint) (*Product, error)
	FindByWaveAndID(ctx context.Context, waveID uint, id uint) (*Product, error)
	ListByWave(ctx context.Context, waveID uint) ([]Product, error)
	FindByWaveAndSKU(ctx context.Context, waveID uint, platform, sku string) (*Product, error)
	DeleteByWave(ctx context.Context, waveID uint) error
}

// CarrierMappingRepository defines persistence operations for CarrierMapping.
type CarrierMappingRepository interface {
	Create(ctx context.Context, mapping *CarrierMapping) error
	Update(ctx context.Context, mapping *CarrierMapping) error
	ListByProfile(ctx context.Context, profileID uint) ([]CarrierMapping, error)
	FindByProfileAndInternal(ctx context.Context, profileID uint, internalCode string) (*CarrierMapping, error)
	FindByProfileAndExternal(ctx context.Context, profileID uint, externalCode string) (*CarrierMapping, error)
	Delete(ctx context.Context, id uint) error
}

// MergeSuggestion represents a suggestion to merge two customer profiles.
type MergeSuggestion struct {
	ID              uint
	SourceProfileID uint
	TargetProfileID uint
	Reason          string
	Status          string
}

// DuplicateGroup holds a grouping key and the comma-separated profile IDs
// that share that key, used by merge-suggestion detection queries.
type DuplicateGroup struct {
	Key        string
	ProfileIDs string
}

// MergeSuggestionRepository defines persistence operations for MergeSuggestion.
type MergeSuggestionRepository interface {
	ListPending(ctx context.Context) ([]MergeSuggestion, error)
	Dismiss(ctx context.Context, id uint) error
	CountBySourceAndTarget(ctx context.Context, sourceID, targetID uint) (int64, error)
	Create(ctx context.Context, suggestion *MergeSuggestion) error
	FindEmailDuplicates(ctx context.Context) ([]DuplicateGroup, error)
	FindPhoneDuplicates(ctx context.Context) ([]DuplicateGroup, error)
}
