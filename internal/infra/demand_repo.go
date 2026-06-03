package infra

import (
	"context"
	"time"

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

func (r *demandRepository) Create(ctx context.Context, doc *domain.DemandDocument) error {
	p := persistence.DemandDocumentFromDomain(doc)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*doc = *persistence.DemandDocumentToDomain(p)
	return nil
}

func (r *demandRepository) FindByID(ctx context.Context, id uint) (*domain.DemandDocument, error) {
	var p persistence.DemandDocument
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.DemandDocumentToDomain(&p), nil
}

func (r *demandRepository) List(ctx context.Context) ([]domain.DemandDocument, error) {
	var ps []persistence.DemandDocument
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.DemandDocumentToDomain(&p)
	}
	return result, nil
}

func (r *demandRepository) CountByIntegrationProfileID(ctx context.Context, profileID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.DemandDocument{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *demandRepository) ListUnassigned(ctx context.Context) ([]domain.DemandDocument, error) {
	var ps []persistence.DemandDocument
	if err := r.db.WithContext(ctx).Where("id NOT IN (?)", r.db.Table("wave_demand_assignments").Select("demand_document_id")).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.DemandDocumentToDomain(&p)
	}
	return result, nil
}

func (r *demandRepository) CreateLine(ctx context.Context, line *domain.DemandLine) error {
	p := persistence.DemandLineFromDomain(line)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.DemandLineToDomain(p)
	return nil
}

func (r *demandRepository) FindLineByID(ctx context.Context, id uint) (*domain.DemandLine, error) {
	var p persistence.DemandLine
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.DemandLineToDomain(&p), nil
}

func (r *demandRepository) ListLinesByDocument(ctx context.Context, docID uint) ([]domain.DemandLine, error) {
	var ps []persistence.DemandLine
	if err := r.db.WithContext(ctx).Where("demand_document_id = ?", docID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.DemandLineToDomain(&p)
	}
	return result, nil
}

func (r *demandRepository) UpdateLine(ctx context.Context, line *domain.DemandLine) error {
	p := persistence.DemandLineFromDomain(line)
	p.ID = line.ID
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *demandRepository) UpdateLineRoutingFields(ctx context.Context, lineID uint, routingDisposition string, recipientInputState string, routingReasonCode string) error {
	return r.db.WithContext(ctx).Model(&persistence.DemandLine{}).Where("id = ?", lineID).Updates(map[string]interface{}{
		"routing_disposition":   routingDisposition,
		"recipient_input_state": recipientInputState,
		"routing_reason_code":   routingReasonCode,
		"updated_at":            time.Now(),
	}).Error
}

func (r *demandRepository) UpdateBoundProfileSnapshot(ctx context.Context, docID uint, snapshot string) error {
	return r.db.WithContext(ctx).Model(&persistence.DemandDocument{}).Where("id = ?", docID).Updates(map[string]interface{}{
		"bound_profile_snapshot": snapshot,
		"updated_at":             time.Now(),
	}).Error
}

func (r *demandRepository) BulkUpdateCustomerProfileID(ctx context.Context, oldProfileID, newProfileID uint) (int64, error) {
	res := r.db.WithContext(ctx).Model(&persistence.DemandDocument{}).Where("customer_profile_id = ?", oldProfileID).Update("customer_profile_id", newProfileID)
	return res.RowsAffected, res.Error
}
