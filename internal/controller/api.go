package controller

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ctx returns the Wails application context captured at startup so backend
// work is cancelled when the app shuts down; tests that never run startup
// fall back to a background context.
func (c *WorkspaceController) ctx() context.Context {
	if appContext == nil {
		return context.Background()
	}
	return appContext
}

func (c *WorkspaceController) EnsureBuiltinPlatforms() error {
	return c.ws.EnsureBuiltinPlatforms(c.ctx())
}

func (c *WorkspaceController) ListPlatforms() ([]domain.Platform, error) {
	return c.ws.ListPlatforms(c.ctx())
}

func (c *WorkspaceController) CreateCustomer(name, notes string) (*domain.CustomerProfile, error) {
	cust := &domain.CustomerProfile{DisplayName: name, Notes: notes}
	if err := c.ws.CreateCustomer(c.ctx(), cust); err != nil {
		return nil, err
	}
	return cust, nil
}

func (c *WorkspaceController) ListCustomers() ([]domain.CustomerProfile, error) {
	return c.ws.ListCustomers(c.ctx())
}

func (c *WorkspaceController) GetCustomer(id uint) (*domain.CustomerProfile, error) {
	return c.ws.GetCustomer(c.ctx(), id)
}

func (c *WorkspaceController) CreateAddress(a domain.RecipientAddress) (*domain.RecipientAddress, error) {
	if err := c.ws.CreateAddress(c.ctx(), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *WorkspaceController) ListAddresses(customerID uint) ([]domain.RecipientAddress, error) {
	return c.ws.ListAddresses(c.ctx(), customerID)
}

func (c *WorkspaceController) CreateProduct(p domain.ProductItem) (*domain.ProductItem, error) {
	if err := c.ws.CreateProduct(c.ctx(), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *WorkspaceController) ListProducts() ([]domain.ProductItem, error) {
	return c.ws.ListProducts(c.ctx())
}

func (c *WorkspaceController) CreateAlias(a domain.ProductAlias) (*domain.ProductAlias, error) {
	if err := c.ws.CreateAlias(c.ctx(), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *WorkspaceController) ListAliases(productID uint) ([]domain.ProductAlias, error) {
	return c.ws.ListAliases(c.ctx(), productID)
}

func (c *WorkspaceController) CreateTemplate(t domain.TemplateConfig) (*domain.TemplateConfig, error) {
	if err := c.ws.CreateTemplate(c.ctx(), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *WorkspaceController) ListTemplates() ([]domain.TemplateConfig, error) {
	return c.ws.ListTemplates(c.ctx())
}

func (c *WorkspaceController) SemanticDictionary() []string { return c.ws.SemanticDictionary() }

func (c *WorkspaceController) NamedTransformers() []string { return c.ws.NamedTransformers() }

func (c *WorkspaceController) CreateCarrierMapping(m domain.CarrierMapping) (*domain.CarrierMapping, error) {
	if err := c.ws.CreateCarrierMapping(c.ctx(), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *WorkspaceController) ListCarrierMappings(platformID uint) ([]domain.CarrierMapping, error) {
	return c.ws.ListCarrierMappings(c.ctx(), platformID)
}

func (c *WorkspaceController) GetSettings() (*domain.AppSettings, error) {
	return c.ws.GetSettings(c.ctx())
}

func (c *WorkspaceController) SaveSettings(s domain.AppSettings) error {
	return c.ws.SaveSettings(c.ctx(), &s)
}

func (c *WorkspaceController) IngestDocument(doc domain.InputDocument, facts []app.IngestFactInput) (*app.IngestDocumentResult, error) {
	document, duplicates, err := c.ws.IngestDocument(c.ctx(), &doc, facts)
	if err != nil {
		return nil, err
	}
	if duplicates == nil {
		duplicates = []domain.DuplicateObservation{}
	}
	return &app.IngestDocumentResult{Document: *document, Duplicates: duplicates}, nil
}

func (c *WorkspaceController) AttachIdentity(identityID, customerID uint) error {
	return c.ws.AttachIdentity(c.ctx(), identityID, customerID)
}

func (c *WorkspaceController) AssignLines(waveID uint, lineIDs []uint) error {
	return c.ws.AssignLines(c.ctx(), waveID, lineIDs)
}

func (c *WorkspaceController) ListInboxRows() ([]app.InboxRow, error) {
	return c.ws.ListInboxRows(c.ctx())
}

func (c *WorkspaceController) CreateWave(name, notes string) (*domain.Wave, error) {
	return c.ws.CreateWave(c.ctx(), name, notes)
}

func (c *WorkspaceController) ListWaves() ([]domain.Wave, error) {
	return c.ws.ListWaves(c.ctx())
}

func (c *WorkspaceController) GetWave(id uint) (*domain.Wave, error) {
	return c.ws.GetWave(c.ctx(), id)
}

func (c *WorkspaceController) CloseWave(id uint, result, note string) error {
	return c.ws.CloseWave(c.ctx(), id, result, note)
}

func (c *WorkspaceController) ReopenWave(id uint) error {
	return c.ws.ReopenWave(c.ctx(), id)
}

func (c *WorkspaceController) UpsertRule(rule domain.EntitlementRule) (*domain.EntitlementRule, error) {
	if err := c.ws.UpsertRule(c.ctx(), &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *WorkspaceController) ListRules(waveID uint) ([]domain.EntitlementRule, error) {
	return c.ws.ListRules(c.ctx(), waveID)
}

func (c *WorkspaceController) CreateGrant(waveID, customerID, productID uint, qty int) (*domain.FulfillmentResult, error) {
	return c.ws.CreateGrant(c.ctx(), waveID, customerID, productID, qty)
}

func (c *WorkspaceController) ListResultViews(waveID uint) ([]app.ResultView, error) {
	return c.ws.ListResultViews(c.ctx(), waveID)
}

func (c *WorkspaceController) ProductTotals(waveID uint) ([]app.ProductTotal, error) {
	return c.ws.ProductTotals(c.ctx(), waveID)
}

func (c *WorkspaceController) SetResultAddress(resultID, addressID uint) error {
	return c.ws.SetResultAddress(c.ctx(), resultID, addressID)
}

func (c *WorkspaceController) GenerateFactoryOrder(waveID, factoryID uint) (*app.GenerateFactoryOrderResult, error) {
	order, lines, err := c.ws.GenerateFactoryOrder(c.ctx(), waveID, factoryID)
	if err != nil {
		return nil, err
	}
	if lines == nil {
		lines = []domain.SupplierOrderLine{}
	}
	return &app.GenerateFactoryOrderResult{Order: *order, Lines: lines}, nil
}

func (c *WorkspaceController) ExportFactoryOrder(orderID uint) (*domain.SupplierOrder, error) {
	return c.ws.ExportFactoryOrder(c.ctx(), orderID)
}

func (c *WorkspaceController) VoidFactoryOrder(orderID uint) error {
	return c.ws.VoidFactoryOrder(c.ctx(), orderID)
}

func (c *WorkspaceController) ListSupplierOrders(waveID uint) ([]domain.SupplierOrder, error) {
	return c.ws.ListSupplierOrders(c.ctx(), waveID)
}

func (c *WorkspaceController) ListSupplierOrderLines(orderID uint) ([]domain.SupplierOrderLine, error) {
	return c.ws.ListSupplierOrderLines(c.ctx(), orderID)
}

func (c *WorkspaceController) ImportShipment(trackingID, trackingNo, carrierCode, carrierName string, qty int) (*domain.Shipment, error) {
	return c.ws.ImportShipment(c.ctx(), trackingID, trackingNo, carrierCode, carrierName, qty)
}

func (c *WorkspaceController) GenerateWritebacks(factID uint) ([]domain.ChannelWritebackItem, error) {
	return c.ws.GenerateWritebacks(c.ctx(), factID)
}

func (c *WorkspaceController) Home() (app.HomeBuckets, error) {
	return c.ws.Home(c.ctx())
}
