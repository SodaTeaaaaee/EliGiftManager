package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type GormStore struct {
	db *gorm.DB
}

var _ domain.Store = (*GormStore)(nil)

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) WithTx(ctx context.Context, fn func(domain.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GormStore{db: tx})
	})
}

func wrapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

func first[T any](db *gorm.DB, dest *T) error {
	err := db.First(dest).Error
	return wrapNotFound(err)
}

func (s *GormStore) CreatePlatform(ctx context.Context, p *domain.Platform) error {
	row := persistence.Platform{Key: p.Key, Name: p.Name, Kind: p.Kind, Notes: p.Notes, ExtraData: p.ExtraData}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*p = platformToDomain(row)
	return nil
}

func (s *GormStore) GetPlatform(ctx context.Context, id uint) (*domain.Platform, error) {
	var row persistence.Platform
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := platformToDomain(row)
	return &d, nil
}

func (s *GormStore) GetPlatformByKey(ctx context.Context, key string) (*domain.Platform, error) {
	var row persistence.Platform
	if err := first(s.db.WithContext(ctx).Where("key = ?", key), &row); err != nil {
		return nil, err
	}
	d := platformToDomain(row)
	return &d, nil
}

func (s *GormStore) ListPlatforms(ctx context.Context) ([]domain.Platform, error) {
	var rows []persistence.Platform
	if err := s.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Platform, 0, len(rows))
	for _, r := range rows {
		out = append(out, platformToDomain(r))
	}
	return out, nil
}

func (s *GormStore) CreateCustomer(ctx context.Context, c *domain.CustomerProfile) error {
	row := persistence.CustomerProfile{DisplayName: c.DisplayName, Notes: c.Notes, ExtraData: c.ExtraData}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*c = customerToDomain(row)
	return nil
}

func (s *GormStore) GetCustomer(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	var row persistence.CustomerProfile
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := customerToDomain(row)
	return &d, nil
}

func (s *GormStore) ListCustomers(ctx context.Context) ([]domain.CustomerProfile, error) {
	var rows []persistence.CustomerProfile
	if err := s.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerProfile, 0, len(rows))
	for _, r := range rows {
		out = append(out, customerToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateCustomer(ctx context.Context, c *domain.CustomerProfile) error {
	return s.db.WithContext(ctx).Model(&persistence.CustomerProfile{}).Where("id = ?", c.ID).Updates(map[string]any{
		"display_name": c.DisplayName,
		"notes":        c.Notes,
		"extra_data":   c.ExtraData,
	}).Error
}

func (s *GormStore) CreateIdentity(ctx context.Context, ident *domain.PlatformIdentity) error {
	row := persistence.PlatformIdentity{
		CustomerProfileID: ident.CustomerProfileID,
		PlatformID:        ident.PlatformID,
		IdentityType:      ident.IdentityType,
		IdentityValue:     ident.IdentityValue,
		NormalizedValue:   ident.NormalizedValue,
		ExtraData:         ident.ExtraData,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*ident = identityToDomain(row)
	return nil
}

func (s *GormStore) GetIdentity(ctx context.Context, id uint) (*domain.PlatformIdentity, error) {
	var row persistence.PlatformIdentity
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := identityToDomain(row)
	return &d, nil
}

func (s *GormStore) FindIdentity(ctx context.Context, platformID uint, identityType, normalized string) (*domain.PlatformIdentity, error) {
	var row persistence.PlatformIdentity
	if err := first(s.db.WithContext(ctx).Where("platform_id = ? AND identity_type = ? AND normalized_value = ?", platformID, identityType, normalized), &row); err != nil {
		return nil, err
	}
	d := identityToDomain(row)
	return &d, nil
}

func (s *GormStore) ListIdentitiesByCustomer(ctx context.Context, customerID uint) ([]domain.PlatformIdentity, error) {
	var rows []persistence.PlatformIdentity
	if err := s.db.WithContext(ctx).Where("customer_profile_id = ?", customerID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PlatformIdentity, 0, len(rows))
	for _, r := range rows {
		out = append(out, identityToDomain(r))
	}
	return out, nil
}

func (s *GormStore) ListUnattachedIdentities(ctx context.Context) ([]domain.PlatformIdentity, error) {
	var rows []persistence.PlatformIdentity
	if err := s.db.WithContext(ctx).Where("customer_profile_id IS NULL").Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PlatformIdentity, 0, len(rows))
	for _, r := range rows {
		out = append(out, identityToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateIdentity(ctx context.Context, ident *domain.PlatformIdentity) error {
	return s.db.WithContext(ctx).Model(&persistence.PlatformIdentity{}).Where("id = ?", ident.ID).Updates(map[string]any{
		"customer_profile_id": ident.CustomerProfileID,
		"identity_value":      ident.IdentityValue,
		"normalized_value":    ident.NormalizedValue,
		"extra_data":          ident.ExtraData,
	}).Error
}

func (s *GormStore) CreateAddress(ctx context.Context, a *domain.RecipientAddress) error {
	row := addressFromDomain(*a)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*a = addressToDomain(row)
	return nil
}

func (s *GormStore) GetAddress(ctx context.Context, id uint) (*domain.RecipientAddress, error) {
	var row persistence.RecipientAddress
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := addressToDomain(row)
	return &d, nil
}

func (s *GormStore) ListAddresses(ctx context.Context, customerID uint) ([]domain.RecipientAddress, error) {
	var rows []persistence.RecipientAddress
	if err := s.db.WithContext(ctx).Where("customer_profile_id = ?", customerID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.RecipientAddress, 0, len(rows))
	for _, r := range rows {
		out = append(out, addressToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateAddress(ctx context.Context, a *domain.RecipientAddress) error {
	row := addressFromDomain(*a)
	row.ID = a.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) ClearDefaultAddresses(ctx context.Context, customerID uint) error {
	return s.db.WithContext(ctx).Model(&persistence.RecipientAddress{}).Where("customer_profile_id = ?", customerID).Update("is_default", false).Error
}

func (s *GormStore) CreateProduct(ctx context.Context, p *domain.ProductItem) error {
	row := persistence.ProductItem{Name: p.Name, FactoryPlatformID: p.FactoryPlatformID, FactorySKU: p.FactorySKU, Notes: p.Notes, ExtraData: p.ExtraData}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*p = productToDomain(row)
	return nil
}

func (s *GormStore) GetProduct(ctx context.Context, id uint) (*domain.ProductItem, error) {
	var row persistence.ProductItem
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := productToDomain(row)
	return &d, nil
}

func (s *GormStore) FindProductByFactorySKU(ctx context.Context, factoryPlatformID uint, sku string) (*domain.ProductItem, error) {
	var row persistence.ProductItem
	if err := first(s.db.WithContext(ctx).Where("factory_platform_id = ? AND factory_sku = ?", factoryPlatformID, sku), &row); err != nil {
		return nil, err
	}
	d := productToDomain(row)
	return &d, nil
}

func (s *GormStore) ListProducts(ctx context.Context) ([]domain.ProductItem, error) {
	var rows []persistence.ProductItem
	if err := s.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProductItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, productToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateProduct(ctx context.Context, p *domain.ProductItem) error {
	return s.db.WithContext(ctx).Model(&persistence.ProductItem{}).Where("id = ?", p.ID).Updates(map[string]any{
		"name": p.Name, "notes": p.Notes, "extra_data": p.ExtraData,
	}).Error
}

func (s *GormStore) CreateAlias(ctx context.Context, a *domain.ProductAlias) error {
	row := persistence.ProductAlias{ProductItemID: a.ProductItemID, PlatformID: a.PlatformID, ExternalProductID: a.ExternalProductID, Title: a.Title, Spec: a.Spec, ExtraData: a.ExtraData}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*a = aliasToDomain(row)
	return nil
}

func (s *GormStore) GetAlias(ctx context.Context, id uint) (*domain.ProductAlias, error) {
	var row persistence.ProductAlias
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := aliasToDomain(row)
	return &d, nil
}

func (s *GormStore) FindAlias(ctx context.Context, platformID uint, externalID string) (*domain.ProductAlias, error) {
	var row persistence.ProductAlias
	if err := first(s.db.WithContext(ctx).Where("platform_id = ? AND external_product_id = ?", platformID, externalID), &row); err != nil {
		return nil, err
	}
	d := aliasToDomain(row)
	return &d, nil
}

func (s *GormStore) ListAliases(ctx context.Context, productID uint) ([]domain.ProductAlias, error) {
	var rows []persistence.ProductAlias
	if err := s.db.WithContext(ctx).Where("product_item_id = ?", productID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProductAlias, 0, len(rows))
	for _, r := range rows {
		out = append(out, aliasToDomain(r))
	}
	return out, nil
}

func (s *GormStore) CreateBundleComponent(ctx context.Context, c *domain.ProductBundleComponent) error {
	row := persistence.ProductBundleComponent{AliasID: c.AliasID, ProductItemID: c.ProductItemID, Quantity: c.Quantity}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	c.ID, c.CreatedAt, c.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) ListBundleComponents(ctx context.Context, aliasID uint) ([]domain.ProductBundleComponent, error) {
	var rows []persistence.ProductBundleComponent
	if err := s.db.WithContext(ctx).Where("alias_id = ?", aliasID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProductBundleComponent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ProductBundleComponent{ID: r.ID, AliasID: r.AliasID, ProductItemID: r.ProductItemID, Quantity: r.Quantity, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *GormStore) CreateTemplate(ctx context.Context, t *domain.TemplateConfig) error {
	row := templateFromDomain(*t)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*t = templateToDomain(row)
	return nil
}

func (s *GormStore) GetTemplate(ctx context.Context, id uint) (*domain.TemplateConfig, error) {
	var row persistence.TemplateConfig
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := templateToDomain(row)
	return &d, nil
}

func (s *GormStore) ListTemplates(ctx context.Context) ([]domain.TemplateConfig, error) {
	var rows []persistence.TemplateConfig
	if err := s.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TemplateConfig, 0, len(rows))
	for _, r := range rows {
		out = append(out, templateToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateTemplate(ctx context.Context, t *domain.TemplateConfig) error {
	row := templateFromDomain(*t)
	row.ID = t.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) CreateCarrierMapping(ctx context.Context, m *domain.CarrierMapping) error {
	row := persistence.CarrierMapping{PlatformID: m.PlatformID, ExternalCode: m.ExternalCode, InternalCode: m.InternalCode, InternalName: m.InternalName}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	m.ID, m.CreatedAt, m.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) ListCarrierMappings(ctx context.Context, platformID uint) ([]domain.CarrierMapping, error) {
	var rows []persistence.CarrierMapping
	if err := s.db.WithContext(ctx).Where("platform_id = ?", platformID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CarrierMapping, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.CarrierMapping{ID: r.ID, PlatformID: r.PlatformID, ExternalCode: r.ExternalCode, InternalCode: r.InternalCode, InternalName: r.InternalName, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *GormStore) CreateDocument(ctx context.Context, d *domain.InputDocument) error {
	row := docFromDomain(*d)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*d = docToDomain(row)
	return nil
}

func (s *GormStore) GetDocument(ctx context.Context, id uint) (*domain.InputDocument, error) {
	var row persistence.InputDocument
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := docToDomain(row)
	return &d, nil
}

func (s *GormStore) ListDocuments(ctx context.Context) ([]domain.InputDocument, error) {
	var rows []persistence.InputDocument
	if err := s.db.WithContext(ctx).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InputDocument, 0, len(rows))
	for _, r := range rows {
		out = append(out, docToDomain(r))
	}
	return out, nil
}

func (s *GormStore) CreateFact(ctx context.Context, f *domain.InputFact) error {
	row := factFromDomain(*f)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*f = factToDomain(row)
	return nil
}

func (s *GormStore) GetFact(ctx context.Context, id uint) (*domain.InputFact, error) {
	var row persistence.InputFact
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := factToDomain(row)
	return &d, nil
}

func (s *GormStore) FindFactByStableID(ctx context.Context, platformID uint, stableID string) (*domain.InputFact, error) {
	var row persistence.InputFact
	if err := first(s.db.WithContext(ctx).Where("platform_id = ? AND stable_external_id = ? AND stable_external_id <> ''", platformID, stableID), &row); err != nil {
		return nil, err
	}
	d := factToDomain(row)
	return &d, nil
}

func (s *GormStore) ListFactsByDocument(ctx context.Context, documentID uint) ([]domain.InputFact, error) {
	var rows []persistence.InputFact
	if err := s.db.WithContext(ctx).Where("document_id = ?", documentID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InputFact, 0, len(rows))
	for _, r := range rows {
		out = append(out, factToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateFact(ctx context.Context, f *domain.InputFact) error {
	row := factFromDomain(*f)
	row.ID = f.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) CreateFactLine(ctx context.Context, l *domain.InputFactLine) error {
	row := lineFromDomain(*l)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*l = lineToDomain(row)
	return nil
}

func (s *GormStore) GetFactLine(ctx context.Context, id uint) (*domain.InputFactLine, error) {
	var row persistence.InputFactLine
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := lineToDomain(row)
	return &d, nil
}

func (s *GormStore) ListFactLines(ctx context.Context, factID uint) ([]domain.InputFactLine, error) {
	var rows []persistence.InputFactLine
	if err := s.db.WithContext(ctx).Where("fact_id = ?", factID).Order("source_line_no, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InputFactLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, lineToDomain(r))
	}
	return out, nil
}

func (s *GormStore) ListUnassignedFactLines(ctx context.Context) ([]domain.InputFactLine, error) {
	var rows []persistence.InputFactLine
	if err := s.db.WithContext(ctx).Where("wave_id IS NULL").Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InputFactLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, lineToDomain(r))
	}
	return out, nil
}

func (s *GormStore) ListFactLinesByWave(ctx context.Context, waveID uint) ([]domain.InputFactLine, error) {
	var rows []persistence.InputFactLine
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InputFactLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, lineToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateFactLine(ctx context.Context, l *domain.InputFactLine) error {
	row := lineFromDomain(*l)
	row.ID = l.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) CreateDuplicate(ctx context.Context, d *domain.DuplicateObservation) error {
	row := persistence.DuplicateObservation{DocumentID: d.DocumentID, ExistingFactID: d.ExistingFactID, Verdict: d.Verdict, Reason: d.Reason, Decided: d.Decided}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	d.ID, d.CreatedAt, d.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) ListOpenDuplicates(ctx context.Context) ([]domain.DuplicateObservation, error) {
	var rows []persistence.DuplicateObservation
	if err := s.db.WithContext(ctx).Where("decided = ? AND verdict = ?", false, string(domain.DuplicateAskOperator)).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DuplicateObservation, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.DuplicateObservation{ID: r.ID, DocumentID: r.DocumentID, ExistingFactID: r.ExistingFactID, Verdict: r.Verdict, Reason: r.Reason, Decided: r.Decided, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *GormStore) UpdateDuplicate(ctx context.Context, d *domain.DuplicateObservation) error {
	return s.db.WithContext(ctx).Model(&persistence.DuplicateObservation{}).Where("id = ?", d.ID).Updates(map[string]any{"decided": d.Decided, "verdict": d.Verdict, "reason": d.Reason}).Error
}

func (s *GormStore) CreateWave(ctx context.Context, w *domain.Wave) error {
	row := waveFromDomain(*w)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*w = waveToDomain(row)
	return nil
}

func (s *GormStore) GetWave(ctx context.Context, id uint) (*domain.Wave, error) {
	var row persistence.Wave
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := waveToDomain(row)
	return &d, nil
}

func (s *GormStore) GetWaveByNo(ctx context.Context, waveNo string) (*domain.Wave, error) {
	var row persistence.Wave
	if err := first(s.db.WithContext(ctx).Where("wave_no = ?", waveNo), &row); err != nil {
		return nil, err
	}
	d := waveToDomain(row)
	return &d, nil
}

func (s *GormStore) ListWaves(ctx context.Context) ([]domain.Wave, error) {
	var rows []persistence.Wave
	if err := s.db.WithContext(ctx).Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Wave, 0, len(rows))
	for _, r := range rows {
		out = append(out, waveToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateWave(ctx context.Context, w *domain.Wave) error {
	row := waveFromDomain(*w)
	row.ID = w.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) NextWaveNo(ctx context.Context) (string, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&persistence.Wave{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("W-%06d", count+1), nil
}

func (s *GormStore) CreateRule(ctx context.Context, r *domain.EntitlementRule) error {
	row, err := ruleFromDomain(*r)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	out, err := ruleToDomain(row)
	if err != nil {
		return err
	}
	*r = out
	return nil
}

func (s *GormStore) GetRule(ctx context.Context, id uint) (*domain.EntitlementRule, error) {
	var row persistence.EntitlementRule
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d, err := ruleToDomain(row)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *GormStore) ListRules(ctx context.Context, waveID uint) ([]domain.EntitlementRule, error) {
	var rows []persistence.EntitlementRule
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.EntitlementRule, 0, len(rows))
	for _, r := range rows {
		d, err := ruleToDomain(r)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *GormStore) UpdateRule(ctx context.Context, r *domain.EntitlementRule) error {
	row, err := ruleFromDomain(*r)
	if err != nil {
		return err
	}
	row.ID = r.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) DeleteRule(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&persistence.EntitlementRule{}, id).Error
}

func (s *GormStore) CreateException(ctx context.Context, e *domain.EntitlementException) error {
	row := persistence.EntitlementException{WaveID: e.WaveID, ProductID: e.ProductID, InstanceID: e.InstanceID, Quantity: e.Quantity, Note: e.Note}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	e.ID, e.CreatedAt, e.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) ListExceptions(ctx context.Context, waveID uint) ([]domain.EntitlementException, error) {
	var rows []persistence.EntitlementException
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.EntitlementException, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.EntitlementException{ID: r.ID, WaveID: r.WaveID, ProductID: r.ProductID, InstanceID: r.InstanceID, Quantity: r.Quantity, Note: r.Note, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *GormStore) DeleteException(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&persistence.EntitlementException{}, id).Error
}

func (s *GormStore) CreateInstance(ctx context.Context, i *domain.EntitlementInstance) error {
	row := instanceFromDomain(*i)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*i = instanceToDomain(row)
	return nil
}

func (s *GormStore) GetInstance(ctx context.Context, id uint) (*domain.EntitlementInstance, error) {
	var row persistence.EntitlementInstance
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := instanceToDomain(row)
	return &d, nil
}

func (s *GormStore) GetInstanceByLine(ctx context.Context, waveID, lineID uint) (*domain.EntitlementInstance, error) {
	var row persistence.EntitlementInstance
	if err := first(s.db.WithContext(ctx).Where("wave_id = ? AND input_fact_line_id = ?", waveID, lineID), &row); err != nil {
		return nil, err
	}
	d := instanceToDomain(row)
	return &d, nil
}

func (s *GormStore) ListInstances(ctx context.Context, waveID uint) ([]domain.EntitlementInstance, error) {
	var rows []persistence.EntitlementInstance
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.EntitlementInstance, 0, len(rows))
	for _, r := range rows {
		out = append(out, instanceToDomain(r))
	}
	return out, nil
}

func (s *GormStore) CreateResult(ctx context.Context, r *domain.FulfillmentResult) error {
	row, err := resultFromDomain(*r)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	out, err := resultToDomain(row)
	if err != nil {
		return err
	}
	*r = out
	return nil
}

func (s *GormStore) GetResult(ctx context.Context, id uint) (*domain.FulfillmentResult, error) {
	var row persistence.FulfillmentResult
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d, err := resultToDomain(row)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *GormStore) ListResults(ctx context.Context, waveID uint) ([]domain.FulfillmentResult, error) {
	var rows []persistence.FulfillmentResult
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentResult, 0, len(rows))
	for _, r := range rows {
		d, err := resultToDomain(r)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *GormStore) ListUnfrozenEntitlementResults(ctx context.Context, waveID uint) ([]domain.FulfillmentResult, error) {
	var rows []persistence.FulfillmentResult
	if err := s.db.WithContext(ctx).Where("wave_id = ? AND frozen = ? AND source_kind = ?", waveID, false, string(domain.SourceEntitlementInstance)).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentResult, 0, len(rows))
	for _, r := range rows {
		d, err := resultToDomain(r)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *GormStore) UpdateResult(ctx context.Context, r *domain.FulfillmentResult) error {
	row, err := resultFromDomain(*r)
	if err != nil {
		return err
	}
	row.ID = r.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) DeleteResult(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&persistence.FulfillmentResult{}, id).Error
}

func (s *GormStore) CreateLink(ctx context.Context, l *domain.ExecutionQuantityLink) error {
	row := persistence.ExecutionQuantityLink{FulfillmentResultID: l.FulfillmentResultID, SupplierOrderLineID: l.SupplierOrderLineID, Quantity: l.Quantity}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	l.ID, l.CreatedAt = row.ID, row.CreatedAt
	return nil
}

func (s *GormStore) ListLinksByResult(ctx context.Context, resultID uint) ([]domain.ExecutionQuantityLink, error) {
	var rows []persistence.ExecutionQuantityLink
	if err := s.db.WithContext(ctx).Where("fulfillment_result_id = ?", resultID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return linksToDomain(rows), nil
}

func (s *GormStore) ListLinksByOrderLine(ctx context.Context, lineID uint) ([]domain.ExecutionQuantityLink, error) {
	var rows []persistence.ExecutionQuantityLink
	if err := s.db.WithContext(ctx).Where("supplier_order_line_id = ?", lineID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return linksToDomain(rows), nil
}

func (s *GormStore) DeleteLinksByOrder(ctx context.Context, orderID uint) error {
	var lineIDs []uint
	if err := s.db.WithContext(ctx).Model(&persistence.SupplierOrderLine{}).Where("supplier_order_id = ?", orderID).Pluck("id", &lineIDs).Error; err != nil {
		return err
	}
	if len(lineIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("supplier_order_line_id IN ?", lineIDs).Delete(&persistence.ExecutionQuantityLink{}).Error
}

func (s *GormStore) CreateSupplierOrder(ctx context.Context, o *domain.SupplierOrder) error {
	row := orderFromDomain(*o)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*o = orderToDomain(row)
	return nil
}

func (s *GormStore) GetSupplierOrder(ctx context.Context, id uint) (*domain.SupplierOrder, error) {
	var row persistence.SupplierOrder
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := orderToDomain(row)
	return &d, nil
}

func (s *GormStore) FindOpenSupplierOrder(ctx context.Context, waveID, factoryID uint) (*domain.SupplierOrder, error) {
	var row persistence.SupplierOrder
	if err := first(s.db.WithContext(ctx).Where("wave_id = ? AND factory_platform_id = ? AND status IN ?", waveID, factoryID, []string{string(domain.SupplierOrderDraft), string(domain.SupplierOrderGenerated), string(domain.SupplierOrderExported)}), &row); err != nil {
		return nil, err
	}
	d := orderToDomain(row)
	return &d, nil
}

func (s *GormStore) ListSupplierOrders(ctx context.Context, waveID uint) ([]domain.SupplierOrder, error) {
	var rows []persistence.SupplierOrder
	if err := s.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SupplierOrder, 0, len(rows))
	for _, r := range rows {
		out = append(out, orderToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateSupplierOrder(ctx context.Context, o *domain.SupplierOrder) error {
	row := orderFromDomain(*o)
	row.ID = o.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) CreateSupplierOrderLine(ctx context.Context, l *domain.SupplierOrderLine) error {
	row := solFromDomain(*l)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*l = solToDomain(row)
	return nil
}

func (s *GormStore) GetSupplierOrderLine(ctx context.Context, id uint) (*domain.SupplierOrderLine, error) {
	var row persistence.SupplierOrderLine
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := solToDomain(row)
	return &d, nil
}

func (s *GormStore) GetSupplierOrderLineByTracking(ctx context.Context, trackingID string) (*domain.SupplierOrderLine, error) {
	var row persistence.SupplierOrderLine
	if err := first(s.db.WithContext(ctx).Where("tracking_id = ?", trackingID), &row); err != nil {
		return nil, err
	}
	d := solToDomain(row)
	return &d, nil
}

func (s *GormStore) ListSupplierOrderLines(ctx context.Context, orderID uint) ([]domain.SupplierOrderLine, error) {
	var rows []persistence.SupplierOrderLine
	if err := s.db.WithContext(ctx).Where("supplier_order_id = ?", orderID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SupplierOrderLine, 0, len(rows))
	for _, r := range rows {
		out = append(out, solToDomain(r))
	}
	return out, nil
}

func (s *GormStore) UpdateSupplierOrderLine(ctx context.Context, l *domain.SupplierOrderLine) error {
	row := solFromDomain(*l)
	row.ID = l.ID
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) RetireTrackingID(ctx context.Context, trackingID string, waveID uint) error {
	return s.db.WithContext(ctx).Create(&persistence.RetiredTrackingID{TrackingID: trackingID, WaveID: waveID, RetiredAt: time.Now()}).Error
}

func (s *GormStore) TrackingIDRetired(ctx context.Context, trackingID string) (bool, error) {
	var n int64
	if err := s.db.WithContext(ctx).Model(&persistence.RetiredTrackingID{}).Where("tracking_id = ?", trackingID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *GormStore) CreateShipment(ctx context.Context, sh *domain.Shipment) error {
	row := persistence.Shipment{TrackingID: sh.TrackingID, CarrierCode: sh.CarrierCode, CarrierName: sh.CarrierName, TrackingNo: sh.TrackingNo, ShippedAt: sh.ShippedAt, Quantity: sh.Quantity, ExtraData: sh.ExtraData}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	sh.ID, sh.CreatedAt, sh.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) GetShipment(ctx context.Context, id uint) (*domain.Shipment, error) {
	var row persistence.Shipment
	if err := first(s.db.WithContext(ctx).Where("id = ?", id), &row); err != nil {
		return nil, err
	}
	d := domain.Shipment{ID: row.ID, TrackingID: row.TrackingID, CarrierCode: row.CarrierCode, CarrierName: row.CarrierName, TrackingNo: row.TrackingNo, ShippedAt: row.ShippedAt, Quantity: row.Quantity, ExtraData: row.ExtraData, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	return &d, nil
}

func (s *GormStore) ListShipmentsByTracking(ctx context.Context, trackingID string) ([]domain.Shipment, error) {
	var rows []persistence.Shipment
	if err := s.db.WithContext(ctx).Where("tracking_id = ?", trackingID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Shipment, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Shipment{ID: r.ID, TrackingID: r.TrackingID, CarrierCode: r.CarrierCode, CarrierName: r.CarrierName, TrackingNo: r.TrackingNo, ShippedAt: r.ShippedAt, Quantity: r.Quantity, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *GormStore) CreateWriteback(ctx context.Context, w *domain.ChannelWritebackItem) error {
	row := persistence.ChannelWritebackItem{InputFactID: w.InputFactID, ShipmentID: w.ShipmentID, TrackingNo: w.TrackingNo, CarrierCode: w.CarrierCode, Status: w.Status, ErrorMessage: w.ErrorMessage, Payload: w.Payload}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	w.ID, w.CreatedAt, w.UpdatedAt = row.ID, row.CreatedAt, row.UpdatedAt
	return nil
}

func (s *GormStore) ListWritebacksByFact(ctx context.Context, factID uint) ([]domain.ChannelWritebackItem, error) {
	var rows []persistence.ChannelWritebackItem
	if err := s.db.WithContext(ctx).Where("input_fact_id = ?", factID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return writebacksToDomain(rows), nil
}

func (s *GormStore) ListFailedWritebacks(ctx context.Context) ([]domain.ChannelWritebackItem, error) {
	var rows []persistence.ChannelWritebackItem
	if err := s.db.WithContext(ctx).Where("status = ?", string(domain.WritebackFailed)).Find(&rows).Error; err != nil {
		return nil, err
	}
	return writebacksToDomain(rows), nil
}

func (s *GormStore) UpdateWriteback(ctx context.Context, w *domain.ChannelWritebackItem) error {
	return s.db.WithContext(ctx).Model(&persistence.ChannelWritebackItem{}).Where("id = ?", w.ID).Updates(map[string]any{"status": w.Status, "error_message": w.ErrorMessage, "payload": w.Payload}).Error
}

func (s *GormStore) GetSettings(ctx context.Context) (*domain.AppSettings, error) {
	var row persistence.AppSettings
	err := s.db.WithContext(ctx).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = persistence.AppSettings{Locale: "zh-CN", Theme: "system", Density: "comfortable", DuplicateRecordMinutes: 10, DuplicateAskDays: 10}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	d := domain.AppSettings{ID: row.ID, Locale: row.Locale, Theme: row.Theme, Density: row.Density, DuplicateRecordMinutes: row.DuplicateRecordMinutes, DuplicateAskDays: row.DuplicateAskDays, UpdatedAt: row.UpdatedAt}
	return &d, nil
}

func (s *GormStore) SaveSettings(ctx context.Context, settings *domain.AppSettings) error {
	row := persistence.AppSettings{ID: settings.ID, Locale: settings.Locale, Theme: settings.Theme, Density: settings.Density, DuplicateRecordMinutes: settings.DuplicateRecordMinutes, DuplicateAskDays: settings.DuplicateAskDays}
	if row.ID == 0 {
		return s.db.WithContext(ctx).Create(&row).Error
	}
	return s.db.WithContext(ctx).Save(&row).Error
}

func platformToDomain(r persistence.Platform) domain.Platform {
	return domain.Platform{ID: r.ID, Key: r.Key, Name: r.Name, Kind: r.Kind, Notes: r.Notes, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func customerToDomain(r persistence.CustomerProfile) domain.CustomerProfile {
	return domain.CustomerProfile{ID: r.ID, DisplayName: r.DisplayName, Notes: r.Notes, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func identityToDomain(r persistence.PlatformIdentity) domain.PlatformIdentity {
	return domain.PlatformIdentity{ID: r.ID, CustomerProfileID: r.CustomerProfileID, PlatformID: r.PlatformID, IdentityType: r.IdentityType, IdentityValue: r.IdentityValue, NormalizedValue: r.NormalizedValue, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func addressFromDomain(a domain.RecipientAddress) persistence.RecipientAddress {
	return persistence.RecipientAddress{ID: a.ID, CustomerProfileID: a.CustomerProfileID, Label: a.Label, RecipientName: a.RecipientName, Phone: a.Phone, Country: a.Country, Province: a.Province, City: a.City, District: a.District, AddressLine1: a.AddressLine1, AddressLine2: a.AddressLine2, PostalCode: a.PostalCode, IsDefault: a.IsDefault, ExtraData: a.ExtraData}
}
func addressToDomain(r persistence.RecipientAddress) domain.RecipientAddress {
	return domain.RecipientAddress{ID: r.ID, CustomerProfileID: r.CustomerProfileID, Label: r.Label, RecipientName: r.RecipientName, Phone: r.Phone, Country: r.Country, Province: r.Province, City: r.City, District: r.District, AddressLine1: r.AddressLine1, AddressLine2: r.AddressLine2, PostalCode: r.PostalCode, IsDefault: r.IsDefault, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func productToDomain(r persistence.ProductItem) domain.ProductItem {
	return domain.ProductItem{ID: r.ID, Name: r.Name, FactoryPlatformID: r.FactoryPlatformID, FactorySKU: r.FactorySKU, Notes: r.Notes, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func aliasToDomain(r persistence.ProductAlias) domain.ProductAlias {
	return domain.ProductAlias{ID: r.ID, ProductItemID: r.ProductItemID, PlatformID: r.PlatformID, ExternalProductID: r.ExternalProductID, Title: r.Title, Spec: r.Spec, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func templateFromDomain(t domain.TemplateConfig) persistence.TemplateConfig {
	return persistence.TemplateConfig{ID: t.ID, PlatformID: t.PlatformID, DocumentType: t.DocumentType, Direction: t.Direction, Name: t.Name, Version: t.Version, Builtin: t.Builtin, MappingJSON: t.MappingJSON, LayoutJSON: t.LayoutJSON, Notes: t.Notes, ExtraData: t.ExtraData}
}
func templateToDomain(r persistence.TemplateConfig) domain.TemplateConfig {
	return domain.TemplateConfig{ID: r.ID, PlatformID: r.PlatformID, DocumentType: r.DocumentType, Direction: r.Direction, Name: r.Name, Version: r.Version, Builtin: r.Builtin, MappingJSON: r.MappingJSON, LayoutJSON: r.LayoutJSON, Notes: r.Notes, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func docFromDomain(d domain.InputDocument) persistence.InputDocument {
	return persistence.InputDocument{ID: d.ID, PlatformID: d.PlatformID, DocumentType: d.DocumentType, Direction: d.Direction, OriginalName: d.OriginalName, RawPayload: d.RawPayload, TemplateID: d.TemplateID, TemplateVersion: d.TemplateVersion, ImportedAt: d.ImportedAt, ExtraData: d.ExtraData}
}
func docToDomain(r persistence.InputDocument) domain.InputDocument {
	return domain.InputDocument{ID: r.ID, PlatformID: r.PlatformID, DocumentType: r.DocumentType, Direction: r.Direction, OriginalName: r.OriginalName, RawPayload: r.RawPayload, TemplateID: r.TemplateID, TemplateVersion: r.TemplateVersion, ImportedAt: r.ImportedAt, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func factFromDomain(f domain.InputFact) persistence.InputFact {
	return persistence.InputFact{ID: f.ID, DocumentID: f.DocumentID, PlatformID: f.PlatformID, Kind: f.Kind, StableExternalID: f.StableExternalID, CustomerProfileID: f.CustomerProfileID, PlatformIdentityID: f.PlatformIdentityID, MembershipLevel: f.MembershipLevel, SourceDocumentNo: f.SourceDocumentNo, SourceCreatedAt: f.SourceCreatedAt, ExtraData: f.ExtraData}
}
func factToDomain(r persistence.InputFact) domain.InputFact {
	return domain.InputFact{ID: r.ID, DocumentID: r.DocumentID, PlatformID: r.PlatformID, Kind: r.Kind, StableExternalID: r.StableExternalID, CustomerProfileID: r.CustomerProfileID, PlatformIdentityID: r.PlatformIdentityID, MembershipLevel: r.MembershipLevel, SourceDocumentNo: r.SourceDocumentNo, SourceCreatedAt: r.SourceCreatedAt, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func lineFromDomain(l domain.InputFactLine) persistence.InputFactLine {
	return persistence.InputFactLine{ID: l.ID, FactID: l.FactID, SourceLineNo: l.SourceLineNo, ExternalSKU: l.ExternalSKU, ExternalTitle: l.ExternalTitle, ExternalSpec: l.ExternalSpec, ProductItemID: l.ProductItemID, Quantity: l.Quantity, WaveID: l.WaveID, ExtraData: l.ExtraData}
}
func lineToDomain(r persistence.InputFactLine) domain.InputFactLine {
	return domain.InputFactLine{ID: r.ID, FactID: r.FactID, SourceLineNo: r.SourceLineNo, ExternalSKU: r.ExternalSKU, ExternalTitle: r.ExternalTitle, ExternalSpec: r.ExternalSpec, ProductItemID: r.ProductItemID, Quantity: r.Quantity, WaveID: r.WaveID, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func waveFromDomain(w domain.Wave) persistence.Wave {
	return persistence.Wave{ID: w.ID, WaveNo: w.WaveNo, Name: w.Name, Notes: w.Notes, CloseResult: w.CloseResult, CloseNote: w.CloseNote, ClosedAt: w.ClosedAt, ReopenedAt: w.ReopenedAt, ExtraData: w.ExtraData}
}
func waveToDomain(r persistence.Wave) domain.Wave {
	return domain.Wave{ID: r.ID, WaveNo: r.WaveNo, Name: r.Name, Notes: r.Notes, CloseResult: r.CloseResult, CloseNote: r.CloseNote, ClosedAt: r.ClosedAt, ReopenedAt: r.ReopenedAt, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func ruleFromDomain(r domain.EntitlementRule) (persistence.EntitlementRule, error) {
	b, err := json.Marshal(r.Selector)
	if err != nil {
		return persistence.EntitlementRule{}, err
	}
	return persistence.EntitlementRule{ID: r.ID, WaveID: r.WaveID, ProductID: r.ProductID, SelectorJSON: string(b), Quantity: r.Quantity, Active: r.Active, ExtraData: r.ExtraData}, nil
}
func ruleToDomain(r persistence.EntitlementRule) (domain.EntitlementRule, error) {
	var sel domain.EntitlementSelector
	if r.SelectorJSON != "" {
		if err := json.Unmarshal([]byte(r.SelectorJSON), &sel); err != nil {
			return domain.EntitlementRule{}, err
		}
	}
	return domain.EntitlementRule{ID: r.ID, WaveID: r.WaveID, ProductID: r.ProductID, Selector: sel, Quantity: r.Quantity, Active: r.Active, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}
func instanceFromDomain(i domain.EntitlementInstance) persistence.EntitlementInstance {
	return persistence.EntitlementInstance{ID: i.ID, WaveID: i.WaveID, InputFactLineID: i.InputFactLineID, CustomerProfileID: i.CustomerProfileID, PlatformIdentityID: i.PlatformIdentityID, MembershipLevel: i.MembershipLevel}
}
func instanceToDomain(r persistence.EntitlementInstance) domain.EntitlementInstance {
	return domain.EntitlementInstance{ID: r.ID, WaveID: r.WaveID, InputFactLineID: r.InputFactLineID, CustomerProfileID: r.CustomerProfileID, PlatformIdentityID: r.PlatformIdentityID, MembershipLevel: r.MembershipLevel, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func resultFromDomain(r domain.FulfillmentResult) (persistence.FulfillmentResult, error) {
	b, err := json.Marshal(r.Address)
	if err != nil {
		return persistence.FulfillmentResult{}, err
	}
	return persistence.FulfillmentResult{ID: r.ID, WaveID: r.WaveID, SourceKind: r.SourceKind, EntitlementInstanceID: r.EntitlementInstanceID, InputFactLineID: r.InputFactLineID, InputFactID: r.InputFactID, CustomerProfileID: r.CustomerProfileID, ProductItemID: r.ProductItemID, Quantity: r.Quantity, AddressJSON: string(b), Frozen: r.Frozen, ExtraData: r.ExtraData}, nil
}
func resultToDomain(r persistence.FulfillmentResult) (domain.FulfillmentResult, error) {
	var addr domain.AddressSnapshot
	if r.AddressJSON != "" {
		if err := json.Unmarshal([]byte(r.AddressJSON), &addr); err != nil {
			return domain.FulfillmentResult{}, err
		}
	}
	return domain.FulfillmentResult{ID: r.ID, WaveID: r.WaveID, SourceKind: r.SourceKind, EntitlementInstanceID: r.EntitlementInstanceID, InputFactLineID: r.InputFactLineID, InputFactID: r.InputFactID, CustomerProfileID: r.CustomerProfileID, ProductItemID: r.ProductItemID, Quantity: r.Quantity, Address: addr, Frozen: r.Frozen, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}
func orderFromDomain(o domain.SupplierOrder) persistence.SupplierOrder {
	return persistence.SupplierOrder{ID: o.ID, WaveID: o.WaveID, FactoryPlatformID: o.FactoryPlatformID, Status: o.Status, ExportedAt: o.ExportedAt, VoidedAt: o.VoidedAt, ExportPayload: o.ExportPayload, ExtraData: o.ExtraData}
}
func orderToDomain(r persistence.SupplierOrder) domain.SupplierOrder {
	return domain.SupplierOrder{ID: r.ID, WaveID: r.WaveID, FactoryPlatformID: r.FactoryPlatformID, Status: r.Status, ExportedAt: r.ExportedAt, VoidedAt: r.VoidedAt, ExportPayload: r.ExportPayload, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func solFromDomain(l domain.SupplierOrderLine) persistence.SupplierOrderLine {
	return persistence.SupplierOrderLine{ID: l.ID, SupplierOrderID: l.SupplierOrderID, ProductItemID: l.ProductItemID, FactorySKU: l.FactorySKU, Quantity: l.Quantity, TrackingID: l.TrackingID, TrackingRetired: l.TrackingRetired, ExtraData: l.ExtraData}
}
func solToDomain(r persistence.SupplierOrderLine) domain.SupplierOrderLine {
	return domain.SupplierOrderLine{ID: r.ID, SupplierOrderID: r.SupplierOrderID, ProductItemID: r.ProductItemID, FactorySKU: r.FactorySKU, Quantity: r.Quantity, TrackingID: r.TrackingID, TrackingRetired: r.TrackingRetired, ExtraData: r.ExtraData, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func linksToDomain(rows []persistence.ExecutionQuantityLink) []domain.ExecutionQuantityLink {
	out := make([]domain.ExecutionQuantityLink, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ExecutionQuantityLink{ID: r.ID, FulfillmentResultID: r.FulfillmentResultID, SupplierOrderLineID: r.SupplierOrderLineID, Quantity: r.Quantity, CreatedAt: r.CreatedAt})
	}
	return out
}
func writebacksToDomain(rows []persistence.ChannelWritebackItem) []domain.ChannelWritebackItem {
	out := make([]domain.ChannelWritebackItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ChannelWritebackItem{ID: r.ID, InputFactID: r.InputFactID, ShipmentID: r.ShipmentID, TrackingNo: r.TrackingNo, CarrierCode: r.CarrierCode, Status: r.Status, ErrorMessage: r.ErrorMessage, Payload: r.Payload, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out
}
