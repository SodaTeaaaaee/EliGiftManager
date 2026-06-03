package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

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

	p := persistence.WaveDemandAssignmentFromDomain(assignment)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("demand document %d is already assigned to wave %d", assignment.DemandDocumentID, assignment.WaveID)
		}
		return err
	}
	*assignment = *persistence.WaveDemandAssignmentToDomain(p)
	return nil
}

func (r *waveDemandAssignmentRepository) DeleteByWaveAndDocument(ctx context.Context, waveID uint, demandDocumentID uint) error {
	return r.db.WithContext(ctx).Where("wave_id = ? AND demand_document_id = ?", waveID, demandDocumentID).
		Delete(&persistence.WaveDemandAssignment{}).Error
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
