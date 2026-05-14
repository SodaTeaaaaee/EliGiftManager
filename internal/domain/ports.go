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
}

// AllocationPolicyRuleRepository defines persistence operations for AllocationPolicyRule.
type AllocationPolicyRuleRepository interface {
	Create(rule *AllocationPolicyRule) error
	FindByID(id uint) (*AllocationPolicyRule, error)
	ListByWave(waveID uint) ([]AllocationPolicyRule, error)
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

	// AtomicCreateShipment creates a shipment and its lines atomically.
	AtomicCreateShipment(shipment *Shipment, lines []*ShipmentLine) error
}

// ChannelSyncRepository defines persistence operations for ChannelSyncJob and ChannelSyncItem.
type ChannelSyncRepository interface {
	CreateJob(job *ChannelSyncJob) error
	FindJobByID(id uint) (*ChannelSyncJob, error)
	ListJobsByWave(waveID uint) ([]ChannelSyncJob, error)

	CreateItem(item *ChannelSyncItem) error
	ListItemsByJob(jobID uint) ([]ChannelSyncItem, error)

	// AtomicCreateChannelSync creates a job and its items atomically.
	AtomicCreateChannelSync(job *ChannelSyncJob, items []*ChannelSyncItem) error
}

// IntegrationProfileRepository defines persistence operations for IntegrationProfile.
type IntegrationProfileRepository interface {
	Create(profile *IntegrationProfile) error
	FindByID(id uint) (*IntegrationProfile, error)
	FindByProfileKey(profileKey string) (*IntegrationProfile, error)
	List() ([]IntegrationProfile, error)
}
