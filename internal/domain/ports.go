package domain

// CustomerProfileRepository defines persistence operations for CustomerProfile and CustomerIdentity.
type CustomerProfileRepository interface {
	Create(profile *CustomerProfile) error
	FindByID(id uint) (*CustomerProfile, error)
	List() ([]CustomerProfile, error)

	CreateIdentity(identity *CustomerIdentity) error
	ListIdentitiesByProfile(profileID uint) ([]CustomerIdentity, error)
}

// DemandDocumentRepository defines persistence operations for DemandDocument and DemandLine.
type DemandDocumentRepository interface {
	Create(doc *DemandDocument) error
	FindByID(id uint) (*DemandDocument, error)
	List() ([]DemandDocument, error)
	ListUnassigned() ([]DemandDocument, error)
	CountByProfileID(profileID uint) (int64, error)

	CreateLine(line *DemandLine) error
	FindLineByID(id uint) (*DemandLine, error)
	ListLinesByDocument(docID uint) ([]DemandLine, error)
}

// WaveRepository defines persistence operations for Wave and WaveParticipantSnapshot.
type WaveRepository interface {
	Create(wave *Wave) error
	FindByID(id uint) (*Wave, error)
	FindByWaveNo(waveNo string) (*Wave, error)
	List() ([]Wave, error)

	AddParticipant(snap *WaveParticipantSnapshot) error
	ListParticipantsByWave(waveID uint) ([]WaveParticipantSnapshot, error)
}

// FulfillmentLineRepository defines persistence operations for FulfillmentLine.
type FulfillmentLineRepository interface {
	Create(line *FulfillmentLine) error
	FindByID(id uint) (*FulfillmentLine, error)
	ListByWave(waveID uint) ([]FulfillmentLine, error)
	DeleteByWaveAndGeneratedBy(waveID uint, generatedBy string) error
	ReplaceByWaveAndGeneratedBy(waveID uint, generatedBy string, newLines []FulfillmentLine) error
}

// SupplierOrderRepository defines persistence operations for SupplierOrder and SupplierOrderLine.
type SupplierOrderRepository interface {
	Create(order *SupplierOrder) error
	FindByID(id uint) (*SupplierOrder, error)
	List() ([]SupplierOrder, error)
	ListByWave(waveID uint) ([]SupplierOrder, error)
	DeleteDraftsByWave(waveID uint) error

	CreateLine(line *SupplierOrderLine) error
	ListLinesByOrder(orderID uint) ([]SupplierOrderLine, error)
	FindLineByID(id uint) (*SupplierOrderLine, error)
	DeleteLinesByOrder(orderID uint) error

	// AtomicCreateSupplierOrder creates order + lines + optional basis pin in one transaction.
	AtomicCreateSupplierOrder(order *SupplierOrder, lines []*SupplierOrderLine, pin *BasisPinParam) error
}

// AllocationPolicyRuleRepository defines persistence operations for AllocationPolicyRule.
type AllocationPolicyRuleRepository interface {
	Create(rule *AllocationPolicyRule) error
	FindByID(id uint) (*AllocationPolicyRule, error)
	ListByWave(waveID uint) ([]AllocationPolicyRule, error)
	Update(rule *AllocationPolicyRule) error
	Delete(id uint) error
}

// WaveDemandAssignmentRepository defines persistence operations for wave-demand linkage.
type WaveDemandAssignmentRepository interface {
	Create(assignment *WaveDemandAssignment) error
	ListByWave(waveID uint) ([]WaveDemandAssignment, error)
	ListByDemandDocument(docID uint) ([]WaveDemandAssignment, error)
	ListDemandDocumentsByWave(waveID uint) ([]DemandDocument, error)
}

// ShipmentRepository defines persistence operations for Shipment and ShipmentLine.
type ShipmentRepository interface {
	Create(shipment *Shipment) error
	FindByID(id uint) (*Shipment, error)
	ListBySupplierOrder(supplierOrderID uint) ([]Shipment, error)
	ListByWave(waveID uint) ([]Shipment, error)

	CreateLine(line *ShipmentLine) error
	ListLinesByShipment(shipmentID uint) ([]ShipmentLine, error)

	// AtomicCreateShipment creates a shipment, its lines, and optional basis pin atomically.
	AtomicCreateShipment(shipment *Shipment, lines []*ShipmentLine, pin *BasisPinParam) error
}

// ChannelSyncRepository defines persistence operations for ChannelSyncJob and ChannelSyncItem.
type ChannelSyncRepository interface {
	CreateJob(job *ChannelSyncJob) error
	FindJobByID(id uint) (*ChannelSyncJob, error)
	ListJobsByWave(waveID uint) ([]ChannelSyncJob, error)
	SaveJob(job *ChannelSyncJob) error

	CreateItem(item *ChannelSyncItem) error
	SaveItem(item *ChannelSyncItem) error
	ListItemsByJob(jobID uint) ([]ChannelSyncItem, error)

	// AtomicCreateChannelSync creates a job, its items, and optional basis pin atomically.
	AtomicCreateChannelSync(job *ChannelSyncJob, items []*ChannelSyncItem, pin *BasisPinParam) error

	CountJobsByProfileID(profileID uint) (int64, error)
}

// ChannelClosureDecisionRepository defines persistence operations for channel closure decision records.
type ChannelClosureDecisionRepository interface {
	Create(record *ChannelClosureDecisionRecord) error
	AtomicCreate(records []*ChannelClosureDecisionRecord) error
	ListByFulfillmentLine(fulfillmentLineID uint) ([]ChannelClosureDecisionRecord, error)
	ListByWave(waveID uint) ([]ChannelClosureDecisionRecord, error)
	CountByProfileID(profileID uint) (int64, error)
}

// IntegrationProfileRepository defines persistence operations for IntegrationProfile.
type IntegrationProfileRepository interface {
	Create(profile *IntegrationProfile) error
	FindByID(id uint) (*IntegrationProfile, error)
	FindByProfileKey(profileKey string) (*IntegrationProfile, error)
	List() ([]IntegrationProfile, error)
	Update(profile *IntegrationProfile) error
	Delete(id uint) error
}

// FulfillmentAdjustmentRepository defines persistence operations for FulfillmentAdjustment.
type FulfillmentAdjustmentRepository interface {
	Create(adj *FulfillmentAdjustment) error
	ListByWave(waveID uint) ([]FulfillmentAdjustment, error)
	ListByFulfillmentLine(fulfillmentLineID uint) ([]FulfillmentAdjustment, error)
}

// DocumentTemplateRepository defines persistence operations for DocumentTemplate.
type DocumentTemplateRepository interface {
	Create(t *DocumentTemplate) error
	FindByID(id uint) (*DocumentTemplate, error)
	FindByKey(key string) (*DocumentTemplate, error)
	List() ([]DocumentTemplate, error)
	ListByDocumentType(docType string) ([]DocumentTemplate, error)
}

// ProfileTemplateBindingRepository defines persistence operations for IntegrationProfileTemplateBinding.
type ProfileTemplateBindingRepository interface {
	Create(b *IntegrationProfileTemplateBinding) error
	ListByProfile(profileID uint) ([]IntegrationProfileTemplateBinding, error)
	FindDefaultByProfileAndType(profileID uint, docType string) (*IntegrationProfileTemplateBinding, error)
	Delete(id uint) error
	CountByProfileID(profileID uint) (int64, error)
}

// HistoryScopeRepository defines persistence operations for HistoryScope.
type HistoryScopeRepository interface {
	Create(scope *HistoryScope) error
	FindByID(id uint) (*HistoryScope, error)
	FindByScopeTypeAndKey(scopeType string, scopeKey string) (*HistoryScope, error)
	UpdateHead(scopeID uint, headNodeID uint) error
}

// HistoryNodeRepository defines persistence operations for HistoryNode.
type HistoryNodeRepository interface {
	Create(node *HistoryNode) error
	FindByID(id uint) (*HistoryNode, error)
	UpdatePreferredRedoChild(nodeID uint, childID uint) error
}

// HistoryCheckpointRepository defines persistence operations for HistoryCheckpoint.
type HistoryCheckpointRepository interface {
	Create(cp *HistoryCheckpoint) error
	FindByNodeID(nodeID uint) (*HistoryCheckpoint, error)
}

// HistoryPinRepository defines persistence operations for HistoryPin.
type HistoryPinRepository interface {
	Create(pin *HistoryPin) error
	ListByNodeID(nodeID uint) ([]HistoryPin, error)
	CountByNodeID(nodeID uint) (int64, error)
}

// ProductMasterRepository defines persistence operations for ProductMaster.
type ProductMasterRepository interface {
	Create(master *ProductMaster) error
	FindByID(id uint) (*ProductMaster, error)
	List() ([]ProductMaster, error)
	FindByPlatformAndSKU(platform, sku string) (*ProductMaster, error)
	Update(master *ProductMaster) error
}

// ProductRepository defines persistence operations for Product.
type ProductRepository interface {
	Create(product *Product) error
	FindByID(id uint) (*Product, error)
	FindByWaveAndID(waveID uint, id uint) (*Product, error)
	ListByWave(waveID uint) ([]Product, error)
	FindByWaveAndSKU(waveID uint, platform, sku string) (*Product, error)
	DeleteByWave(waveID uint) error
}
