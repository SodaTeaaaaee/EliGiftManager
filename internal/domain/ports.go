package domain

import (
	"context"
	"time"
)

type Store interface {
	WithTx(ctx context.Context, fn func(Store) error) error

	CreatePlatform(ctx context.Context, p *Platform) error
	GetPlatform(ctx context.Context, id uint) (*Platform, error)
	GetPlatformByKey(ctx context.Context, key string) (*Platform, error)
	ListPlatforms(ctx context.Context) ([]Platform, error)

	CreateCustomer(ctx context.Context, c *CustomerProfile) error
	GetCustomer(ctx context.Context, id uint) (*CustomerProfile, error)
	ListCustomers(ctx context.Context) ([]CustomerProfile, error)
	UpdateCustomer(ctx context.Context, c *CustomerProfile) error

	CreateIdentity(ctx context.Context, ident *PlatformIdentity) error
	GetIdentity(ctx context.Context, id uint) (*PlatformIdentity, error)
	FindIdentity(ctx context.Context, platformID uint, identityType, normalized string) (*PlatformIdentity, error)
	ListIdentitiesByCustomer(ctx context.Context, customerID uint) ([]PlatformIdentity, error)
	ListUnattachedIdentities(ctx context.Context) ([]PlatformIdentity, error)
	UpdateIdentity(ctx context.Context, ident *PlatformIdentity) error

	CreateAddress(ctx context.Context, a *RecipientAddress) error
	GetAddress(ctx context.Context, id uint) (*RecipientAddress, error)
	ListAddresses(ctx context.Context, customerID uint) ([]RecipientAddress, error)
	UpdateAddress(ctx context.Context, a *RecipientAddress) error
	ClearDefaultAddresses(ctx context.Context, customerID uint) error

	CreateProduct(ctx context.Context, p *ProductItem) error
	GetProduct(ctx context.Context, id uint) (*ProductItem, error)
	FindProductByFactorySKU(ctx context.Context, factoryPlatformID uint, sku string) (*ProductItem, error)
	ListProducts(ctx context.Context) ([]ProductItem, error)
	UpdateProduct(ctx context.Context, p *ProductItem) error

	CreateAlias(ctx context.Context, a *ProductAlias) error
	GetAlias(ctx context.Context, id uint) (*ProductAlias, error)
	FindAlias(ctx context.Context, platformID uint, externalID string) (*ProductAlias, error)
	ListAliases(ctx context.Context, productID uint) ([]ProductAlias, error)
	UpdateAlias(ctx context.Context, a *ProductAlias) error

	CreateBundleComponent(ctx context.Context, c *ProductBundleComponent) error
	ListBundleComponents(ctx context.Context, aliasID uint) ([]ProductBundleComponent, error)

	CreateTemplate(ctx context.Context, t *TemplateConfig) error
	GetTemplate(ctx context.Context, id uint) (*TemplateConfig, error)
	ListTemplates(ctx context.Context) ([]TemplateConfig, error)
	UpdateTemplate(ctx context.Context, t *TemplateConfig) error

	CreateCarrierMapping(ctx context.Context, m *CarrierMapping) error
	ListCarrierMappings(ctx context.Context, platformID uint) ([]CarrierMapping, error)

	CreateDocument(ctx context.Context, d *InputDocument) error
	GetDocument(ctx context.Context, id uint) (*InputDocument, error)
	ListDocuments(ctx context.Context) ([]InputDocument, error)

	CreateFact(ctx context.Context, f *InputFact) error
	GetFact(ctx context.Context, id uint) (*InputFact, error)
	FindFactByStableID(ctx context.Context, platformID uint, stableID string) (*InputFact, error)
	ListFactsByDocument(ctx context.Context, documentID uint) ([]InputFact, error)
	UpdateFact(ctx context.Context, f *InputFact) error

	CreateFactLine(ctx context.Context, l *InputFactLine) error
	GetFactLine(ctx context.Context, id uint) (*InputFactLine, error)
	ListFactLines(ctx context.Context, factID uint) ([]InputFactLine, error)
	ListUnassignedFactLines(ctx context.Context) ([]InputFactLine, error)
	ListFactLinesByWave(ctx context.Context, waveID uint) ([]InputFactLine, error)
	ListFactLinesByExternalSKU(ctx context.Context, platformID uint, sku string) ([]InputFactLine, error)
	UpdateFactLine(ctx context.Context, l *InputFactLine) error

	CreateDuplicate(ctx context.Context, d *DuplicateObservation) error
	ListOpenDuplicates(ctx context.Context) ([]DuplicateObservation, error)
	UpdateDuplicate(ctx context.Context, d *DuplicateObservation) error

	CreateWave(ctx context.Context, w *Wave) error
	GetWave(ctx context.Context, id uint) (*Wave, error)
	GetWaveByNo(ctx context.Context, waveNo string) (*Wave, error)
	ListWaves(ctx context.Context) ([]Wave, error)
	UpdateWave(ctx context.Context, w *Wave) error
	NextWaveNo(ctx context.Context) (string, error)

	CreateRule(ctx context.Context, r *EntitlementRule) error
	GetRule(ctx context.Context, id uint) (*EntitlementRule, error)
	ListRules(ctx context.Context, waveID uint) ([]EntitlementRule, error)
	UpdateRule(ctx context.Context, r *EntitlementRule) error
	DeleteRule(ctx context.Context, id uint) error

	CreateException(ctx context.Context, e *EntitlementException) error
	ListExceptions(ctx context.Context, waveID uint) ([]EntitlementException, error)
	DeleteException(ctx context.Context, id uint) error

	CreateInstance(ctx context.Context, i *EntitlementInstance) error
	GetInstance(ctx context.Context, id uint) (*EntitlementInstance, error)
	GetInstanceByLine(ctx context.Context, waveID, lineID uint) (*EntitlementInstance, error)
	ListInstances(ctx context.Context, waveID uint) ([]EntitlementInstance, error)

	CreateResult(ctx context.Context, r *FulfillmentResult) error
	GetResult(ctx context.Context, id uint) (*FulfillmentResult, error)
	ListResults(ctx context.Context, waveID uint) ([]FulfillmentResult, error)
	ListUnfrozenEntitlementResults(ctx context.Context, waveID uint) ([]FulfillmentResult, error)
	UpdateResult(ctx context.Context, r *FulfillmentResult) error
	DeleteResult(ctx context.Context, id uint) error

	CreateLink(ctx context.Context, l *ExecutionQuantityLink) error
	ListLinksByResult(ctx context.Context, resultID uint) ([]ExecutionQuantityLink, error)
	ListLinksByOrderLine(ctx context.Context, lineID uint) ([]ExecutionQuantityLink, error)
	DeleteLinksByOrder(ctx context.Context, orderID uint) error

	CreateSupplierOrder(ctx context.Context, o *SupplierOrder) error
	GetSupplierOrder(ctx context.Context, id uint) (*SupplierOrder, error)
	FindOpenSupplierOrder(ctx context.Context, waveID, factoryID uint) (*SupplierOrder, error)
	ListSupplierOrders(ctx context.Context, waveID uint) ([]SupplierOrder, error)
	UpdateSupplierOrder(ctx context.Context, o *SupplierOrder) error

	CreateSupplierOrderLine(ctx context.Context, l *SupplierOrderLine) error
	GetSupplierOrderLine(ctx context.Context, id uint) (*SupplierOrderLine, error)
	GetSupplierOrderLineByTracking(ctx context.Context, trackingID string) (*SupplierOrderLine, error)
	ListSupplierOrderLines(ctx context.Context, orderID uint) ([]SupplierOrderLine, error)
	UpdateSupplierOrderLine(ctx context.Context, l *SupplierOrderLine) error

	RetireTrackingID(ctx context.Context, trackingID string, waveID uint) error
	TrackingIDRetired(ctx context.Context, trackingID string) (bool, error)

	CreateShipment(ctx context.Context, s *Shipment) error
	GetShipment(ctx context.Context, id uint) (*Shipment, error)
	ListShipmentsByTracking(ctx context.Context, trackingID string) ([]Shipment, error)
	UpdateShipmentShippedAt(ctx context.Context, id uint, shippedAt *time.Time) error

	CreateWriteback(ctx context.Context, w *ChannelWritebackItem) error
	ListWritebacksByFact(ctx context.Context, factID uint) ([]ChannelWritebackItem, error)
	ListFailedWritebacks(ctx context.Context) ([]ChannelWritebackItem, error)
	UpdateWriteback(ctx context.Context, w *ChannelWritebackItem) error

	GetSettings(ctx context.Context) (*AppSettings, error)
	SaveSettings(ctx context.Context, s *AppSettings) error
}
