package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func (ws *Workspace) EnsureBuiltinPlatforms(ctx context.Context) error {
	if _, err := ws.Store.GetPlatformByKey(ctx, "bilibili"); err != nil {
		if err := ws.Store.CreatePlatform(ctx, &domain.Platform{Key: "bilibili", Name: "哔哩哔哩", Kind: string(domain.PlatformKindSource)}); err != nil {
			return err
		}
	}
	if _, err := ws.Store.GetPlatformByKey(ctx, "rozao"); err != nil {
		if err := ws.Store.CreatePlatform(ctx, &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}); err != nil {
			return err
		}
	}
	_, err := ws.Store.GetSettings(ctx)
	return err
}

func (ws *Workspace) CreatePlatform(ctx context.Context, p *domain.Platform) error {
	return ws.Store.CreatePlatform(ctx, p)
}

func (ws *Workspace) ListPlatforms(ctx context.Context) ([]domain.Platform, error) {
	return ws.Store.ListPlatforms(ctx)
}

func (ws *Workspace) CreateCustomer(ctx context.Context, c *domain.CustomerProfile) error {
	if c.DisplayName == "" {
		return fmt.Errorf("customer display name is required")
	}
	return ws.Store.CreateCustomer(ctx, c)
}

func (ws *Workspace) ListCustomers(ctx context.Context) ([]domain.CustomerProfile, error) {
	return ws.Store.ListCustomers(ctx)
}

func (ws *Workspace) GetCustomer(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	return ws.Store.GetCustomer(ctx, id)
}

func (ws *Workspace) UpdateCustomer(ctx context.Context, c *domain.CustomerProfile) error {
	return ws.Store.UpdateCustomer(ctx, c)
}

func (ws *Workspace) CreateAddress(ctx context.Context, a *domain.RecipientAddress) error {
	if a.IsDefault {
		if err := ws.Store.ClearDefaultAddresses(ctx, a.CustomerProfileID); err != nil {
			return err
		}
	}
	return ws.Store.CreateAddress(ctx, a)
}

func (ws *Workspace) ListAddresses(ctx context.Context, customerID uint) ([]domain.RecipientAddress, error) {
	return ws.Store.ListAddresses(ctx, customerID)
}

func (ws *Workspace) CreateProduct(ctx context.Context, p *domain.ProductItem) error {
	if p.FactorySKU == "" || p.FactoryPlatformID == 0 {
		return fmt.Errorf("product requires factory platform and sku")
	}
	return ws.Store.CreateProduct(ctx, p)
}

func (ws *Workspace) ListProducts(ctx context.Context) ([]domain.ProductItem, error) {
	return ws.Store.ListProducts(ctx)
}

func (ws *Workspace) CreateAlias(ctx context.Context, a *domain.ProductAlias) error {
	return ws.Store.CreateAlias(ctx, a)
}

func (ws *Workspace) ListAliases(ctx context.Context, productID uint) ([]domain.ProductAlias, error) {
	return ws.Store.ListAliases(ctx, productID)
}

func (ws *Workspace) CreateBundleComponent(ctx context.Context, c *domain.ProductBundleComponent) error {
	return ws.Store.CreateBundleComponent(ctx, c)
}

func (ws *Workspace) CreateTemplate(ctx context.Context, t *domain.TemplateConfig) error {
	if t.Version == 0 {
		t.Version = 1
	}
	return ws.Store.CreateTemplate(ctx, t)
}

func (ws *Workspace) ListTemplates(ctx context.Context) ([]domain.TemplateConfig, error) {
	return ws.Store.ListTemplates(ctx)
}

func (ws *Workspace) SemanticDictionary() []string {
	return append([]string(nil), domain.SemanticDictionary...)
}

func (ws *Workspace) NamedTransformers() []string {
	return append([]string(nil), domain.NamedTransformers...)
}

func (ws *Workspace) CreateCarrierMapping(ctx context.Context, m *domain.CarrierMapping) error {
	return ws.Store.CreateCarrierMapping(ctx, m)
}

func (ws *Workspace) ListCarrierMappings(ctx context.Context, platformID uint) ([]domain.CarrierMapping, error) {
	return ws.Store.ListCarrierMappings(ctx, platformID)
}

func (ws *Workspace) GetSettings(ctx context.Context) (*domain.AppSettings, error) {
	return ws.Store.GetSettings(ctx)
}

func (ws *Workspace) SaveSettings(ctx context.Context, s *domain.AppSettings) error {
	if s.DuplicateRecordMinutes <= 0 {
		s.DuplicateRecordMinutes = 10
	}
	if s.DuplicateAskDays <= 0 {
		s.DuplicateAskDays = 10
	}
	return ws.Store.SaveSettings(ctx, s)
}
