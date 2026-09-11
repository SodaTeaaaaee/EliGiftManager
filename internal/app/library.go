package app

import (
	"context"
	"fmt"
	"strings"

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

// UpdateAlias re-points an alias at another unified product and re-runs the
// alignment this alias drives: fact lines carrying the alias's external SKU
// move to the new product, their plain retail results follow in place
// (addresses stay), and bundle-expanded results are rebuilt from the new
// alias's components. Lines whose results are already frozen by a factory
// order keep the old alignment — execution history is not rewritten.
func (ws *Workspace) UpdateAlias(ctx context.Context, aliasID, productItemID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		alias, err := tx.GetAlias(ctx, aliasID)
		if err != nil {
			return err
		}
		if _, err := tx.GetProduct(ctx, productItemID); err != nil {
			return fmt.Errorf("update alias: target product: %w", err)
		}
		if alias.ProductItemID == productItemID {
			return nil
		}
		alias.ProductItemID = productItemID
		if err := tx.UpdateAlias(ctx, alias); err != nil {
			return err
		}
		return tws.realignAliasLines(ctx, alias)
	})
}

// realignAliasLines re-runs alignment for the lines an alias resolves.
func (ws *Workspace) realignAliasLines(ctx context.Context, alias *domain.ProductAlias) error {
	lines, err := ws.Store.ListFactLinesByExternalSKU(ctx, alias.PlatformID, alias.ExternalProductID)
	if err != nil {
		return err
	}
	waves, err := ws.Store.ListWaves(ctx)
	if err != nil {
		return err
	}
	for i := range lines {
		line := &lines[i]
		if line.ProductItemID != nil && *line.ProductItemID == alias.ProductItemID {
			continue
		}
		fact, err := ws.Store.GetFact(ctx, line.FactID)
		if err != nil {
			return err
		}
		rebuilt := false
		for _, w := range waves {
			results, err := ws.Store.ListResults(ctx, w.ID)
			if err != nil {
				return err
			}
			for j := range results {
				r := &results[j]
				if r.InputFactLineID == nil || *r.InputFactLineID != line.ID || r.SourceKind != string(domain.SourceRetailLine) {
					continue
				}
				if r.Frozen {
					// Frozen by a factory order: execution wins over alignment.
					continue
				}
				if strings.Contains(r.ExtraData, bundleAliasLineMarker) {
					// Bundle composition changed with the alias: rebuild.
					if err := ws.Store.DeleteResult(ctx, r.ID); err != nil {
						return err
					}
					rebuilt = true
					continue
				}
				pid := alias.ProductItemID
				r.ProductItemID = &pid
				if err := ws.Store.UpdateResult(ctx, r); err != nil {
					return err
				}
			}
		}
		if line.WaveID != nil && rebuilt {
			if err := ws.ensureRetailResult(ctx, *line.WaveID, fact, line); err != nil {
				return err
			}
		}
		pid := alias.ProductItemID
		line.ProductItemID = &pid
		if err := ws.Store.UpdateFactLine(ctx, line); err != nil {
			return err
		}
	}
	return nil
}

func (ws *Workspace) CreateBundleComponent(ctx context.Context, c *domain.ProductBundleComponent) error {
	return ws.Store.CreateBundleComponent(ctx, c)
}

func (ws *Workspace) SemanticDictionary() []string {
	return append([]string(nil), domain.SemanticDictionary...)
}

func (ws *Workspace) NamedTransformers() []string {
	return append([]string(nil), domain.NamedTransformers...)
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
