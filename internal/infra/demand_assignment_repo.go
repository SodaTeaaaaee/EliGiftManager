package infra

import (
	"fmt"
	"strings"

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

func (r *waveDemandAssignmentRepository) Create(assignment *domain.WaveDemandAssignment) error {
	// Check cross-wave duplicate: current phase does not support assigning the same demand to multiple waves
	existing, err := r.ListByDemandDocument(assignment.DemandDocumentID)
	if err != nil {
		return err
	}
	for _, a := range existing {
		if a.WaveID != assignment.WaveID {
			return fmt.Errorf("demand document %d is already assigned to wave %d; cross-wave assignment is not supported in the current phase", assignment.DemandDocumentID, a.WaveID)
		}
	}

	p := persistence.ToPersistenceWaveDemandAssignment(assignment)
	if err := r.db.Create(p).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "idx_wave_demand") {
			return fmt.Errorf("demand document %d is already assigned to wave %d", assignment.DemandDocumentID, assignment.WaveID)
		}
		return err
	}
	*assignment = *persistence.FromPersistenceWaveDemandAssignment(p)
	return nil
}

func (r *waveDemandAssignmentRepository) ListByWave(waveID uint) ([]domain.WaveDemandAssignment, error) {
	var ps []persistence.WaveDemandAssignment
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveDemandAssignment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWaveDemandAssignment(&p)
	}
	return result, nil
}

func (r *waveDemandAssignmentRepository) ListByDemandDocument(docID uint) ([]domain.WaveDemandAssignment, error) {
	var ps []persistence.WaveDemandAssignment
	if err := r.db.Where("demand_document_id = ?", docID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveDemandAssignment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWaveDemandAssignment(&p)
	}
	return result, nil
}

func (r *waveDemandAssignmentRepository) ListDemandDocumentsByWave(waveID uint) ([]domain.DemandDocument, error) {
	var assignments []persistence.WaveDemandAssignment
	if err := r.db.Where("wave_id = ?", waveID).Find(&assignments).Error; err != nil {
		return nil, err
	}
	if len(assignments) == 0 {
		return nil, nil
	}

	ids := make([]uint, len(assignments))
	for i, a := range assignments {
		ids[i] = a.DemandDocumentID
	}

	var ps []persistence.DemandDocument
	if err := r.db.Where("id IN ?", ids).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandDocument, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceDemandDocument(&p)
	}
	return result, nil
}
