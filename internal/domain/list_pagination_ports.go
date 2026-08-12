package domain

import "context"

type ListPageQuery struct {
	SortBy  string
	SortDir string
	Limit   int
	Offset  int
}

type CustomerProfilePageQuery struct {
	ListPageQuery
	Keyword            string
	Platform           string
	MissingAddressOnly bool
}

type ProductMasterPageQuery struct {
	ListPageQuery
	Keyword      string
	ProductKinds []string
	ArchivedOnly bool
}

type DemandInboxPageQuery struct {
	ListPageQuery
	Assignment           string
	DemandKind           string
	IntegrationProfileID *uint
	WaveID               *uint
}

type ShipmentByWavePageQuery struct {
	ListPageQuery
	WaveID uint
}

type CustomerProfilePageRepository interface {
	ListCustomerProfilesPage(context.Context, CustomerProfilePageQuery) ([]CustomerProfile, int64, error)
	ListCustomerIdentitiesByProfileIDs(context.Context, []uint) ([]CustomerIdentity, error)
	ListCustomerAddressesByProfileIDs(context.Context, []uint) ([]CustomerAddress, error)
	FindMatchedCustomerHistoricalNames(context.Context, []uint, string) (map[uint]string, error)
	ListCustomerIdentityPlatforms(context.Context) ([]string, error)
}

type ProductMasterPageRepository interface {
	ListProductMastersPage(context.Context, ProductMasterPageQuery) ([]ProductMaster, int64, error)
}

type DemandInboxPageRepository interface {
	ListDemandDocumentsPage(context.Context, DemandInboxPageQuery) ([]DemandDocument, int64, error)
	ListDemandAssignmentsByDocumentIDs(context.Context, []uint) ([]WaveDemandAssignment, error)
	ListDemandLinesByDocumentIDs(context.Context, []uint) ([]DemandLine, error)
}

type ShipmentPageRepository interface {
	ListShipmentsPage(context.Context, ShipmentByWavePageQuery) ([]Shipment, int64, error)
	ListShipmentLinesByShipmentIDs(context.Context, []uint) ([]ShipmentLine, error)
}
