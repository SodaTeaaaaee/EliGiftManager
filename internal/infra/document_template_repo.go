package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *documentTemplateRepository) Create(ctx context.Context, t *domain.DocumentTemplate) error {
	p := persistence.DocumentTemplateFromDomain(t)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	t.ID = p.ID
	t.CreatedAt = p.CreatedAt
	t.UpdatedAt = p.UpdatedAt
	return nil
}

func (r *documentTemplateRepository) FindByID(ctx context.Context, id uint) (*domain.DocumentTemplate, error) {
	var p persistence.DocumentTemplate
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.DocumentTemplateToDomain(&p), nil
}

func (r *documentTemplateRepository) FindByKey(ctx context.Context, key string) (*domain.DocumentTemplate, error) {
	var p persistence.DocumentTemplate
	if err := r.db.WithContext(ctx).Where("template_key = ?", key).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.DocumentTemplateToDomain(&p), nil
}

func (r *documentTemplateRepository) List(ctx context.Context) ([]domain.DocumentTemplate, error) {
	var records []persistence.DocumentTemplate
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DocumentTemplate, len(records))
	for i := range records {
		out[i] = *persistence.DocumentTemplateToDomain(&records[i])
	}
	return out, nil
}

func (r *documentTemplateRepository) ListByDocumentType(ctx context.Context, docType string) ([]domain.DocumentTemplate, error) {
	var records []persistence.DocumentTemplate
	if err := r.db.WithContext(ctx).Where("document_type = ?", docType).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DocumentTemplate, len(records))
	for i := range records {
		out[i] = *persistence.DocumentTemplateToDomain(&records[i])
	}
	return out, nil
}

func (r *documentTemplateRepository) Update(ctx context.Context, t *domain.DocumentTemplate) error {
	p := persistence.DocumentTemplateFromDomain(t)
	p.ID = t.ID
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*t = *persistence.DocumentTemplateToDomain(p)
	return nil
}

func (r *documentTemplateRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tmpl persistence.DocumentTemplate
		if err := tx.First(&tmpl, id).Error; err != nil {
			return err
		}
		// SQLite unique indexes include soft-deleted rows. Rewrite the key before
		// soft deletion so operators may recreate the same stable template key.
		deletedKey := fmt.Sprintf("%s__deleted_%d_%d", tmpl.TemplateKey, id, time.Now().UTC().UnixNano())
		if err := tx.Model(&tmpl).Update("template_key", deletedKey).Error; err != nil {
			return err
		}
		return tx.Delete(&tmpl).Error
	})
}
