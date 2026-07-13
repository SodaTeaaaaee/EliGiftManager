package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// NewDemandInboxAssignmentRepository returns the bulk wave-demand-assignment lookup
// capability used by the demand inbox query (ListDemandInboxRows). It shares the same
// underlying struct as NewWaveDemandAssignmentRepository — only the exposed interface differs.
func NewDemandInboxAssignmentRepository(db *gorm.DB) domain.DemandInboxAssignmentRepository {
	return &waveDemandAssignmentRepository{db: db}
}

// ListByDemandDocumentIDs bulk-fetches wave-demand assignments for a batch of documents in a
// single query, replacing what was previously one ListByDemandDocument call per document.
func (r *waveDemandAssignmentRepository) ListByDemandDocumentIDs(ctx context.Context, docIDs []uint) ([]domain.WaveDemandAssignment, error) {
	if len(docIDs) == 0 {
		return []domain.WaveDemandAssignment{}, nil
	}
	var ps []persistence.WaveDemandAssignment
	if err := r.db.WithContext(ctx).Where("demand_document_id IN ?", docIDs).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveDemandAssignment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveDemandAssignmentToDomain(&p)
	}
	return result, nil
}

// NewDemandInboxLineRepository returns the bulk demand-line lookup capability used by the
// demand inbox query (ListDemandInboxRows). It shares the same underlying struct as
// NewDemandRepository — only the exposed interface differs.
func NewDemandInboxLineRepository(db *gorm.DB) domain.DemandInboxLineRepository {
	return &demandRepository{db: db}
}

// ListLinesByDocumentIDs bulk-fetches demand lines for a batch of documents in a single query,
// replacing what was previously one ListLinesByDocument call per document.
func (r *demandRepository) ListLinesByDocumentIDs(ctx context.Context, docIDs []uint) ([]domain.DemandLine, error) {
	if len(docIDs) == 0 {
		return []domain.DemandLine{}, nil
	}
	var ps []persistence.DemandLine
	if err := r.db.WithContext(ctx).Where("demand_document_id IN ?", docIDs).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DemandLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.DemandLineToDomain(&p)
	}
	return result, nil
}
