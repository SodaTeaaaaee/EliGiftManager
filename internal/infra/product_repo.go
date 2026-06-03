package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// ---- ProductMasterRepository ----

type productMasterRepository struct {
	db *gorm.DB
}

func NewProductMasterRepository(db *gorm.DB) domain.ProductMasterRepository {
	return &productMasterRepository{db: db}
}

func (r *productMasterRepository) Create(ctx context.Context, master *domain.ProductMaster) error {
	p := persistence.ProductMasterFromDomain(master)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*master = *persistence.ProductMasterToDomain(p)
	return nil
}

func (r *productMasterRepository) FindByID(ctx context.Context, id uint) (*domain.ProductMaster, error) {
	var p persistence.ProductMaster
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.ProductMasterToDomain(&p), nil
}

func (r *productMasterRepository) List(ctx context.Context) ([]domain.ProductMaster, error) {
	var ps []persistence.ProductMaster
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ProductMaster, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ProductMasterToDomain(&p)
	}
	return result, nil
}

func (r *productMasterRepository) FindByPlatformAndSKU(ctx context.Context, platform, sku string) (*domain.ProductMaster, error) {
	var p persistence.ProductMaster
	if err := r.db.WithContext(ctx).Where("supplier_platform = ? AND factory_sku = ?", platform, sku).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.ProductMasterToDomain(&p), nil
}

func (r *productMasterRepository) Update(ctx context.Context, master *domain.ProductMaster) error {
	p := persistence.ProductMasterFromDomain(master)
	p.ID = master.ID
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*master = *persistence.ProductMasterToDomain(p)
	return nil
}

// ---- ProductRepository ----

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	p := persistence.ProductFromDomain(product)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*product = *persistence.ProductToDomain(p)
	return nil
}

func (r *productRepository) FindByID(ctx context.Context, id uint) (*domain.Product, error) {
	var p persistence.Product
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.ProductToDomain(&p), nil
}

func (r *productRepository) FindByWaveAndID(ctx context.Context, waveID uint, id uint) (*domain.Product, error) {
	var p persistence.Product
	if err := r.db.WithContext(ctx).Where("wave_id = ? AND id = ?", waveID, id).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.ProductToDomain(&p), nil
}

func (r *productRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.Product, error) {
	var ps []persistence.Product
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Product, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ProductToDomain(&p)
	}
	return result, nil
}

func (r *productRepository) FindByWaveAndSKU(ctx context.Context, waveID uint, platform, sku string) (*domain.Product, error) {
	var p persistence.Product
	if err := r.db.WithContext(ctx).Where("wave_id = ? AND supplier_platform = ? AND factory_sku = ?", waveID, platform, sku).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.ProductToDomain(&p), nil
}

func (r *productRepository) DeleteByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Where("wave_id = ?", waveID).Delete(&persistence.Product{}).Error
}
