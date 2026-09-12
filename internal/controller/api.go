package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func (c *WorkspaceController) EnsureBuiltinPlatforms() error {
	return c.ws.EnsureBuiltinPlatforms(c.ctx)
}

// EnsureBuiltinTemplates re-seeds the read-only template catalog; startup
// runs it after EnsureBuiltinPlatforms.
func (c *WorkspaceController) EnsureBuiltinTemplates() error {
	return c.ws.EnsureBuiltinTemplates(c.ctx)
}

func (c *WorkspaceController) ListPlatforms() ([]domain.Platform, error) {
	return c.ws.ListPlatforms(c.ctx)
}

func (c *WorkspaceController) CreatePlatform(p domain.Platform) error {
	return c.ws.CreatePlatform(c.ctx, &p)
}

func (c *WorkspaceController) CreateCustomer(name, notes string) (*domain.CustomerProfile, error) {
	cust := &domain.CustomerProfile{DisplayName: name, Notes: notes}
	if err := c.ws.CreateCustomer(c.ctx, cust); err != nil {
		return nil, err
	}
	return cust, nil
}

func (c *WorkspaceController) ListCustomers() ([]domain.CustomerProfile, error) {
	return c.ws.ListCustomers(c.ctx)
}

func (c *WorkspaceController) GetCustomer(id uint) (*domain.CustomerProfile, error) {
	return c.ws.GetCustomer(c.ctx, id)
}

func (c *WorkspaceController) UpdateCustomer(cust domain.CustomerProfile) error {
	return c.ws.UpdateCustomer(c.ctx, &cust)
}

func (c *WorkspaceController) CreateAddress(a domain.RecipientAddress) (*domain.RecipientAddress, error) {
	if err := c.ws.CreateAddress(c.ctx, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *WorkspaceController) ListAddresses(customerID uint) ([]domain.RecipientAddress, error) {
	return c.ws.ListAddresses(c.ctx, customerID)
}

func (c *WorkspaceController) CreateProduct(p domain.ProductItem) (*domain.ProductItem, error) {
	if err := c.ws.CreateProduct(c.ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *WorkspaceController) ListProducts() ([]domain.ProductItem, error) {
	return c.ws.ListProducts(c.ctx)
}

func (c *WorkspaceController) CreateAlias(a domain.ProductAlias) (*domain.ProductAlias, error) {
	if err := c.ws.CreateAlias(c.ctx, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *WorkspaceController) ListAliases(productID uint) ([]domain.ProductAlias, error) {
	return c.ws.ListAliases(c.ctx, productID)
}

func (c *WorkspaceController) CreateBundleComponent(comp domain.ProductBundleComponent) error {
	return c.ws.CreateBundleComponent(c.ctx, &comp)
}

// CreateTemplate stores a new active (user) template at version 1.
func (c *WorkspaceController) CreateTemplate(t domain.TemplateConfig) (*domain.TemplateConfig, error) {
	if err := c.ws.CreateTemplate(c.ctx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *WorkspaceController) GetTemplate(id uint) (*domain.TemplateConfig, error) {
	return c.ws.GetTemplate(c.ctx, id)
}

// UpdateTemplate edits a user template in place and advances its version;
// built-in rows are refused.
func (c *WorkspaceController) UpdateTemplate(t domain.TemplateConfig) (*domain.TemplateConfig, error) {
	return c.ws.UpdateTemplate(c.ctx, &t)
}

// DeleteTemplate removes a user template; built-in rows are refused.
func (c *WorkspaceController) DeleteTemplate(id uint) error {
	return c.ws.DeleteTemplate(c.ctx, id)
}

// ListTemplates returns built-in and user templates; the Builtin flag tells
// them apart.
func (c *WorkspaceController) ListTemplates() ([]domain.TemplateConfig, error) {
	return c.ws.ListTemplates(c.ctx)
}

// DocumentTypeCatalog returns the closed document-type set with the direction
// and platform kind each type locks.
func (c *WorkspaceController) DocumentTypeCatalog() []app.DocumentTypeInfo {
	return c.ws.DocumentTypeCatalog()
}

func (c *WorkspaceController) SemanticDictionary() []string { return c.ws.SemanticDictionary() }

func (c *WorkspaceController) NamedTransformers() []string { return c.ws.NamedTransformers() }

func (c *WorkspaceController) CreateCarrierMapping(m domain.CarrierMapping) (*domain.CarrierMapping, error) {
	if err := c.ws.CreateCarrierMapping(c.ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *WorkspaceController) ListCarrierMappings(platformID uint) ([]domain.CarrierMapping, error) {
	return c.ws.ListCarrierMappings(c.ctx, platformID)
}

func (c *WorkspaceController) UpdateCarrierMapping(m domain.CarrierMapping) (*domain.CarrierMapping, error) {
	return c.ws.UpdateCarrierMapping(c.ctx, &m)
}

func (c *WorkspaceController) DeleteCarrierMapping(id uint) error {
	return c.ws.DeleteCarrierMapping(c.ctx, id)
}

// ImportCarrierMappings loads a source platform's carrier table from a file;
// nameHeader and codeHeader name the columns carrying the carrier description
// and the platform's carrier code.
func (c *WorkspaceController) ImportCarrierMappings(platformID uint, filePath, nameHeader, codeHeader string) (*app.ImportCarrierMappingsResult, error) {
	return c.ws.ImportCarrierMappings(c.ctx, platformID, filePath, nameHeader, codeHeader)
}

func (c *WorkspaceController) UpdateAlias(aliasID, productItemID uint) error {
	return c.ws.UpdateAlias(c.ctx, aliasID, productItemID)
}

func (c *WorkspaceController) ImportFile(platformID, templateID uint, filePath string) (*app.ImportFileResult, error) {
	return c.ws.ImportFile(c.ctx, platformID, templateID, filePath)
}

// InspectSampleFile returns a file's raw records (first row = candidate
// header), its format, and xlsx sheet names, without applying any mapping.
func (c *WorkspaceController) InspectSampleFile(filePath, sheetName string, limit int) (*app.SampleFileInfo, error) {
	return c.ws.InspectSampleFile(c.ctx, filePath, sheetName, limit)
}

// PreviewMapping runs an unsaved mapping JSON against a sample file.
func (c *WorkspaceController) PreviewMapping(mappingJSON, documentType, filePath string, limit int) (*alignment.TemplatePreview, error) {
	return c.ws.PreviewMapping(c.ctx, mappingJSON, documentType, filePath, limit)
}

// PreviewTemplate runs a saved template (built-in ones included) against a
// sample file.
func (c *WorkspaceController) PreviewTemplate(templateID uint, filePath string, limit int) (*alignment.TemplatePreview, error) {
	return c.ws.PreviewTemplate(c.ctx, templateID, filePath, limit)
}

func (c *WorkspaceController) GetSettings() (*domain.AppSettings, error) {
	return c.ws.GetSettings(c.ctx)
}

func (c *WorkspaceController) SaveSettings(s domain.AppSettings) error {
	return c.ws.SaveSettings(c.ctx, &s)
}

func (c *WorkspaceController) IngestDocument(doc domain.InputDocument, facts []app.IngestFactInput) (*app.IngestDocumentResult, error) {
	document, duplicates, err := c.ws.IngestDocument(c.ctx, &doc, facts)
	if err != nil {
		return nil, err
	}
	if duplicates == nil {
		duplicates = []domain.DuplicateObservation{}
	}
	return &app.IngestDocumentResult{Document: *document, Duplicates: duplicates}, nil
}

func (c *WorkspaceController) AttachIdentity(identityID, customerID uint) error {
	return c.ws.AttachIdentity(c.ctx, identityID, customerID)
}

func (c *WorkspaceController) AssignLines(waveID uint, lineIDs []uint) error {
	return c.ws.AssignLines(c.ctx, waveID, lineIDs)
}

func (c *WorkspaceController) MoveLines(lineIDs []uint, targetWaveID uint) error {
	return c.ws.MoveLines(c.ctx, lineIDs, targetWaveID)
}

func (c *WorkspaceController) ApplyRevision(factID uint) error {
	return c.ws.ApplyRevision(c.ctx, factID)
}

func (c *WorkspaceController) DismissRevision(factID uint) error {
	return c.ws.DismissRevision(c.ctx, factID)
}

func (c *WorkspaceController) DecideDuplicate(id uint, accept bool) error {
	return c.ws.DecideDuplicate(c.ctx, id, accept)
}

func (c *WorkspaceController) ListInboxRows() ([]app.InboxRow, error) {
	return c.ws.ListInboxRows(c.ctx)
}

func (c *WorkspaceController) CreateWave(name, notes string) (*domain.Wave, error) {
	return c.ws.CreateWave(c.ctx, name, notes)
}

func (c *WorkspaceController) ListWaves() ([]domain.Wave, error) {
	return c.ws.ListWaves(c.ctx)
}

func (c *WorkspaceController) GetWave(id uint) (*domain.Wave, error) {
	return c.ws.GetWave(c.ctx, id)
}

func (c *WorkspaceController) CloseWave(id uint, result, note string) error {
	return c.ws.CloseWave(c.ctx, id, result, note)
}

func (c *WorkspaceController) ReopenWave(id uint) error {
	return c.ws.ReopenWave(c.ctx, id)
}

func (c *WorkspaceController) UpsertRule(rule domain.EntitlementRule) (*domain.EntitlementRule, error) {
	if err := c.ws.UpsertRule(c.ctx, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *WorkspaceController) ListRules(waveID uint) ([]domain.EntitlementRule, error) {
	return c.ws.ListRules(c.ctx, waveID)
}

func (c *WorkspaceController) DeleteRule(id uint) error {
	return c.ws.DeleteRule(c.ctx, id)
}

func (c *WorkspaceController) AddException(e domain.EntitlementException) error {
	return c.ws.AddException(c.ctx, &e)
}

func (c *WorkspaceController) DeleteException(id uint) error {
	return c.ws.DeleteException(c.ctx, id)
}

// ListExceptions returns the wave's entitlement exceptions joined with
// customer and product display names.
func (c *WorkspaceController) ListExceptions(waveID uint) ([]app.ExceptionView, error) {
	return c.ws.ListExceptions(c.ctx, waveID)
}

// ListEntitlementInstances returns the wave's membership instances with
// customer name and platform identity summary.
func (c *WorkspaceController) ListEntitlementInstances(waveID uint) ([]app.InstanceView, error) {
	return c.ws.ListEntitlementInstances(c.ctx, waveID)
}

func (c *WorkspaceController) UpsertQuantitySplitRule(rule domain.QuantitySplitRule) error {
	return c.ws.UpsertQuantitySplitRule(c.ctx, &rule)
}

func (c *WorkspaceController) DeleteQuantitySplitRule(id uint) error {
	return c.ws.DeleteQuantitySplitRule(c.ctx, id)
}

func (c *WorkspaceController) ListQuantitySplitRules(waveID uint) ([]domain.QuantitySplitRule, error) {
	return c.ws.ListQuantitySplitRules(c.ctx, waveID)
}

func (c *WorkspaceController) CreateGrant(waveID, customerID, productID uint, qty int) (*domain.FulfillmentResult, error) {
	return c.ws.CreateGrant(c.ctx, waveID, customerID, productID, qty)
}

func (c *WorkspaceController) ListResultViews(waveID uint) ([]app.ResultView, error) {
	return c.ws.ListResultViews(c.ctx, waveID)
}

func (c *WorkspaceController) ProductTotals(waveID uint) ([]app.ProductTotal, error) {
	return c.ws.ProductTotals(c.ctx, waveID)
}

func (c *WorkspaceController) SetResultAddress(resultID, addressID uint) error {
	return c.ws.SetResultAddress(c.ctx, resultID, addressID)
}

func (c *WorkspaceController) GenerateFactoryOrder(waveID, factoryID uint) (*app.GenerateFactoryOrderResult, error) {
	order, lines, err := c.ws.GenerateFactoryOrder(c.ctx, waveID, factoryID)
	if err != nil {
		return nil, err
	}
	if lines == nil {
		lines = []domain.SupplierOrderLine{}
	}
	return &app.GenerateFactoryOrderResult{Order: *order, Lines: lines}, nil
}

// GenerateFactoryOrderForResults submits only the selected fulfillment
// results; resultIDs nil means the whole wave, an explicitly empty selection
// submits nothing (ErrNothingToSubmit).
func (c *WorkspaceController) GenerateFactoryOrderForResults(waveID, factoryID uint, resultIDs []uint) (*app.GenerateFactoryOrderResult, error) {
	order, lines, err := c.ws.GenerateFactoryOrderForResults(c.ctx, waveID, factoryID, resultIDs)
	if err != nil {
		return nil, err
	}
	if lines == nil {
		lines = []domain.SupplierOrderLine{}
	}
	return &app.GenerateFactoryOrderResult{Order: *order, Lines: lines}, nil
}

func (c *WorkspaceController) ExportFactoryOrder(orderID uint) (*domain.SupplierOrder, error) {
	return c.ws.ExportFactoryOrder(c.ctx, orderID)
}

func (c *WorkspaceController) VoidFactoryOrder(orderID uint) error {
	return c.ws.VoidFactoryOrder(c.ctx, orderID)
}

func (c *WorkspaceController) ListSupplierOrders(waveID uint) ([]domain.SupplierOrder, error) {
	return c.ws.ListSupplierOrders(c.ctx, waveID)
}

func (c *WorkspaceController) ListSupplierOrderLines(orderID uint) ([]domain.SupplierOrderLine, error) {
	return c.ws.ListSupplierOrderLines(c.ctx, orderID)
}

func (c *WorkspaceController) ImportShipment(trackingID, trackingNo, carrierCode, carrierName string, qty int) (*domain.Shipment, error) {
	return c.ws.ImportShipment(c.ctx, trackingID, trackingNo, carrierCode, carrierName, qty)
}

// ImportShipmentFile imports a factory shipment return through the given
// active shipment_return template of the factory platform.
func (c *WorkspaceController) ImportShipmentFile(platformID, templateID uint, filePath string) (*app.ImportShipmentFileResult, error) {
	return c.ws.ImportShipmentFile(c.ctx, platformID, templateID, filePath)
}

func (c *WorkspaceController) ExportFactoryOrderFile(orderID uint) (*app.ExportFileResult, error) {
	return c.ws.ExportFactoryOrderFile(c.ctx, orderID)
}

func (c *WorkspaceController) GenerateWritebacks(factID uint) ([]domain.ChannelWritebackItem, error) {
	return c.ws.GenerateWritebacks(c.ctx, factID)
}

// MarkWritebackSent records a successful channel writeback for one parcel.
func (c *WorkspaceController) MarkWritebackSent(writebackID uint) error {
	return c.ws.MarkWritebackSent(c.ctx, writebackID)
}

// MarkWritebackFailed records a failed channel writeback attempt for one
// parcel; the retry counter advances and the error text is kept (truncated).
func (c *WorkspaceController) MarkWritebackFailed(writebackID uint, errMsg string) error {
	return c.ws.MarkWritebackFailed(c.ctx, writebackID, errMsg)
}

// ListWritebacksByWave returns the writeback items behind the wave's facts,
// ordered by item id.
func (c *WorkspaceController) ListWritebacksByWave(waveID uint) ([]domain.ChannelWritebackItem, error) {
	items, err := c.ws.ListWritebacksByWave(c.ctx, waveID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.ChannelWritebackItem{}
	}
	return items, nil
}

// ExportWritebackFile writes one writeback item's stored payload to the
// exports directory and returns the absolute path.
func (c *WorkspaceController) ExportWritebackFile(writebackID uint) (*app.WritebackFileResult, error) {
	return c.ws.ExportWritebackFile(c.ctx, writebackID)
}

func (c *WorkspaceController) Home() (app.HomeBuckets, error) {
	return c.ws.Home(c.ctx)
}
