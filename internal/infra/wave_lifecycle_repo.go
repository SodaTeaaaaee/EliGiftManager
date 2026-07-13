package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// NewWaveLifecycleRepository returns a domain.WaveLifecycleRepository backed by the
// same underlying waveRepository concrete type used by NewWaveRepository
// (internal/infra/wave_repo.go), so both interfaces share one GORM-backed
// implementation without duplicating connection wiring.
func NewWaveLifecycleRepository(db *gorm.DB) domain.WaveLifecycleRepository {
	return &waveRepository{db: db}
}

// UpdateWaveFields persists the operator-editable name/notes/levelTags fields.
// Uses a map (not a struct) so zero-value strings (e.g. clearing notes) are still
// written — GORM's struct-based Updates silently skips zero values.
func (r *waveRepository) UpdateWaveFields(ctx context.Context, waveID uint, name, notes, levelTags string) error {
	return r.db.WithContext(ctx).Model(&persistence.Wave{}).Where("id = ?", waveID).Updates(map[string]interface{}{
		"name":       name,
		"notes":      notes,
		"level_tags": levelTags,
	}).Error
}

// TransitionLifecycleStage sets lifecycle_stage only, leaving progress_snapshot
// untouched.
func (r *waveRepository) TransitionLifecycleStage(ctx context.Context, waveID uint, stage string) error {
	return r.db.WithContext(ctx).Model(&persistence.Wave{}).Where("id = ?", waveID).
		Update("lifecycle_stage", stage).Error
}
