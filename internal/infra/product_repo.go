package infra

import (
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

func (r *productMasterRepository) Create(master *domain.ProductMaster) error {
	p := persistence.ToPersistenceProductMaster(master)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*master = *persistence.FromPersistenceProductMaster(p)
	return nil
}

func (r *productMasterRepository) FindByID(id uint) (*domain.ProductMaster, error) {
	var p persistence.ProductMaster
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceProductMaster(&p), nil
}

func (r *productMasterRepository) List() ([]domain.ProductMaster, error) {
	var ps []persistence.ProductMaster
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ProductMaster, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceProductMaster(&p)
	}
	return result, nil
}

func (r *productMasterRepository) FindByPlatformAndSKU(platform, sku string) (*domain.ProductMaster, error) {
	var p persistence.ProductMaster
	if err := r.db.Where("supplier_platform = ? AND factory_sku = ?", platform, sku).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceProductMaster(&p), nil
}

func (r *productMasterRepository) Update(master *domain.ProductMaster) error {
	p := persistence.ToPersistenceProductMaster(master)
	p.ID = master.ID
	if err := r.db.Save(p).Error; err != nil {
		return err
	}
	*master = *persistence.FromPersistenceProductMaster(p)
	return nil
}

// ---- ProductRepository ----

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(product *domain.Product) error {
	p := persistence.ToPersistenceProduct(product)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*product = *persistence.FromPersistenceProduct(p)
	return nil
}

func (r *productRepository) FindByID(id uint) (*domain.Product, error) {
	var p persistence.Product
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceProduct(&p), nil
}

func (r *productRepository) ListByWave(waveID uint) ([]domain.Product, error) {
	var ps []persistence.Product
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Product, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceProduct(&p)
	}
	return result, nil
}

func (r *productRepository) FindByWaveAndSKU(waveID uint, platform, sku string) (*domain.Product, error) {
	var p persistence.Product
	if err := r.db.Where("wave_id = ? AND supplier_platform = ? AND factory_sku = ?", waveID, platform, sku).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceProduct(&p), nil
}

func (r *productRepository) DeleteByWave(waveID uint) error {
	return r.db.Where("wave_id = ?", waveID).Delete(&persistence.Product{}).Error
}
