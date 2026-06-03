package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

// TxRepos bundles all repository instances bound to a single transaction.
// Created via NewTxRepos inside a gorm.DB.Transaction callback so every
// repository operates on the same transaction scope.
type TxRepos struct {
	RuleRepo          domain.AllocationPolicyRuleRepository
	FulfillRepo       domain.FulfillmentLineRepository
	WaveRepo          domain.WaveRepository
	AdjustmentRepo    domain.FulfillmentAdjustmentRepository
	DemandRepo        domain.DemandDocumentRepository
	AssignmentRepo    domain.WaveDemandAssignmentRepository
	ProductRepo       domain.ProductRepository
	ShipmentRepo      domain.ShipmentRepository
	SupplierRepo      domain.SupplierOrderRepository
	HistoryScope      domain.HistoryScopeRepository
	HistoryNode       domain.HistoryNodeRepository
	HistoryCheckpoint domain.HistoryCheckpointRepository
	HistoryPin        domain.HistoryPinRepository
	ClosureDecision   domain.ChannelClosureDecisionRepository
	ChannelSync       domain.ChannelSyncRepository
	CustomerProfile   domain.CustomerProfileRepository
	Profile           domain.IntegrationProfileRepository
	Address           domain.CustomerAddressRepository
	Binding           domain.ProfileTemplateBindingRepository
	Mapping           domain.CarrierMappingRepository
}

// NewTxRepos creates a TxRepos with every repository constructor invoked
// against the given *gorm.DB (expected to be a transaction handle).
func NewTxRepos(tx *gorm.DB) *TxRepos {
	return &TxRepos{
		RuleRepo:          NewRuleRepository(tx),
		FulfillRepo:       NewFulfillmentRepository(tx),
		WaveRepo:          NewWaveRepository(tx),
		AdjustmentRepo:    NewFulfillmentAdjustmentRepository(tx),
		DemandRepo:        NewDemandRepository(tx),
		AssignmentRepo:    NewWaveDemandAssignmentRepository(tx),
		ProductRepo:       NewProductRepository(tx),
		ShipmentRepo:      NewShipmentRepository(tx),
		SupplierRepo:      NewSupplierOrderRepository(tx),
		HistoryScope:      NewHistoryScopeRepository(tx),
		HistoryNode:       NewHistoryNodeRepository(tx),
		HistoryCheckpoint: NewHistoryCheckpointRepository(tx),
		HistoryPin:        NewHistoryPinRepository(tx),
		ClosureDecision:   NewClosureDecisionRepository(tx),
		ChannelSync:       NewChannelSyncRepository(tx),
		CustomerProfile:   NewProfileRepository(tx),
		Profile:           NewIntegrationProfileRepository(tx),
		Address:           NewAddressRepository(tx),
		Binding:           NewProfileTemplateBindingRepository(tx),
		Mapping:           NewCarrierMappingRepository(tx),
	}
}
