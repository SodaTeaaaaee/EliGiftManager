package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type demandRepository struct {
	db *gorm.DB
}

func NewDemandRepository(db *gorm.DB) domain.DemandDocumentRepository {
	return &demandRepository{db: db}
}

func (r *demandRepository) Create(doc *domain.DemandDocument) error {
	p := persistence.ToPersistenceDemandDocument(doc)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*doc = *persistence.FromPersistenceDemandDocument(p)
	return nil
}

func (r *demandRepository) FindByID(id uint) (*domain.DemandDocument, error) {
	var p persistence.DemandDocument
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceDemandDocument(&p), nil
}

func (r *demandRepository) List() ([]domain.DemandDocument, error) {
	var ps []persistence.DemandDocument
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceDemandDocument(&p)
	}
	return result, nil
}

func (r *demandRepository) CountByProfileID(profileID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&persistence.DemandDocument{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *demandRepository) ListUnassigned() ([]domain.DemandDocument, error) {
	var ps []persistence.DemandDocument
	if err := r.db.Where("id NOT IN (?)", r.db.Table("wave_demand_assignments").Select("demand_document_id")).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceDemandDocument(&p)
	}
	return result, nil
}

func (r *demandRepository) CreateLine(line *domain.DemandLine) error {
	p := persistence.ToPersistenceDemandLine(line)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.FromPersistenceDemandLine(p)
	return nil
}

func (r *demandRepository) FindLineByID(id uint) (*domain.DemandLine, error) {
	var p persistence.DemandLine
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceDemandLine(&p), nil
}

func (r *demandRepository) ListLinesByDocument(docID uint) ([]domain.DemandLine, error) {
	var ps []persistence.DemandLine
	if err := r.db.Where("demand_document_id = ?", docID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceDemandLine(&p)
	}
	return result, nil
}
