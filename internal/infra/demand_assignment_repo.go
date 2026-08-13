package infra

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func isDuplicateAssignmentErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicated key")
}

type waveDemandAssignmentRepository struct {
	db *gorm.DB
}

func NewWaveDemandAssignmentRepository(db *gorm.DB) domain.WaveDemandAssignmentRepository {
	return &waveDemandAssignmentRepository{db: db}
}

func (r *waveDemandAssignmentRepository) Create(ctx context.Context, assignment *domain.WaveDemandAssignment) error {
	// Check cross-wave duplicate: current phase does not support assigning the same demand to multiple waves
	existing, err := r.ListByDemandDocument(ctx, assignment.DemandDocumentID)
	if err != nil {
		return err
	}
	for _, a := range existing {
		if a.WaveID != assignment.WaveID {
			return fmt.Errorf("demand document %d is already assigned to wave %d; cross-wave assignment is not supported in the current phase", assignment.DemandDocumentID, a.WaveID)
		}
	}

	// SQLite unique indexes include soft-deleted rows. Purge leftover
	// soft-deleted residue for this pair so a later re-assign does not collide.
	if err := r.db.WithContext(ctx).Unscoped().
		Where("wave_id = ? AND demand_document_id = ? AND deleted_at IS NOT NULL", assignment.WaveID, assignment.DemandDocumentID).
		Delete(&persistence.WaveDemandAssignment{}).Error; err != nil {
		return err
	}

	p := persistence.WaveDemandAssignmentFromDomain(assignment)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		if isDuplicateAssignmentErr(err) {
			return fmt.Errorf("demand document %d is already assigned to wave %d", assignment.DemandDocumentID, assignment.WaveID)
		}
		return err
	}
	*assignment = *persistence.WaveDemandAssignmentToDomain(p)
	return nil
}

func (r *waveDemandAssignmentRepository) ExistsByDocument(ctx context.Context, demandDocumentID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.WaveDemandAssignment{}).
		Where("demand_document_id = ?", demandDocumentID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *waveDemandAssignmentRepository) DeleteByWaveAndDocument(ctx context.Context, waveID uint, demandDocumentID uint) error {
	// Hard-delete so idx_wave_demand does not keep occupying the unique slot
	// after unassign (same Unscoped treatment as the undo path).
	res := r.db.WithContext(ctx).Unscoped().
		Where("wave_id = ? AND demand_document_id = ?", waveID, demandDocumentID).
		Delete(&persistence.WaveDemandAssignment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("demand document %d is not assigned to wave %d; nothing to unassign", demandDocumentID, waveID)
	}
	return nil
}

func (r *waveDemandAssignmentRepository) DeleteByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("wave_id = ?", waveID).Delete(&persistence.WaveDemandAssignment{}).Error
}

func (r *waveDemandAssignmentRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.WaveDemandAssignment, error) {
	var ps []persistence.WaveDemandAssignment
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveDemandAssignment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveDemandAssignmentToDomain(&p)
	}
	return result, nil
}

func (r *waveDemandAssignmentRepository) ListByDemandDocument(ctx context.Context, docID uint) ([]domain.WaveDemandAssignment, error) {
	var ps []persistence.WaveDemandAssignment
	if err := r.db.WithContext(ctx).Where("demand_document_id = ?", docID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveDemandAssignment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveDemandAssignmentToDomain(&p)
	}
	return result, nil
}

func (r *waveDemandAssignmentRepository) ListDemandDocumentsByWave(ctx context.Context, waveID uint) ([]domain.DemandDocument, error) {
	var assignments []persistence.WaveDemandAssignment
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&assignments).Error; err != nil {
		return nil, err
	}
	if len(assignments) == 0 {
		return []domain.DemandDocument{}, nil
	}

	ids := make([]uint, len(assignments))
	for i, a := range assignments {
		ids[i] = a.DemandDocumentID
	}

	var ps []persistence.DemandDocument
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.DemandDocumentToDomain(&p)
	}
	return result, nil
}
