package infra

import (
	"errors"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type documentTemplateRepository struct {
	db *gorm.DB
}

func NewDocumentTemplateRepository(db *gorm.DB) domain.DocumentTemplateRepository {
	return &documentTemplateRepository{db: db}
}

func (r *documentTemplateRepository) Create(t *domain.DocumentTemplate) error {
	p := persistence.DocumentTemplateFromDomain(t)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	t.ID = p.ID
	t.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	t.UpdatedAt = p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *documentTemplateRepository) FindByID(id uint) (*domain.DocumentTemplate, error) {
	var p persistence.DocumentTemplate
	if err := r.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.DocumentTemplateToDomain(&p), nil
}

func (r *documentTemplateRepository) FindByKey(key string) (*domain.DocumentTemplate, error) {
	var p persistence.DocumentTemplate
	if err := r.db.Where("template_key = ?", key).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.DocumentTemplateToDomain(&p), nil
}

func (r *documentTemplateRepository) List() ([]domain.DocumentTemplate, error) {
	var records []persistence.DocumentTemplate
	if err := r.db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DocumentTemplate, len(records))
	for i := range records {
		out[i] = *persistence.DocumentTemplateToDomain(&records[i])
	}
	return out, nil
}

func (r *documentTemplateRepository) ListByDocumentType(docType string) ([]domain.DocumentTemplate, error) {
	var records []persistence.DocumentTemplate
	if err := r.db.Where("document_type = ?", docType).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DocumentTemplate, len(records))
	for i := range records {
		out[i] = *persistence.DocumentTemplateToDomain(&records[i])
	}
	return out, nil
}
