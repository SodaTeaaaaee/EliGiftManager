package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type waveRepository struct {
	db *gorm.DB
}

func NewWaveRepository(db *gorm.DB) domain.WaveRepository {
	return &waveRepository{db: db}
}

func (r *waveRepository) Create(ctx context.Context, wave *domain.Wave) error {
	p := persistence.WaveFromDomain(wave)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*wave = *persistence.WaveToDomain(p)
	return nil
}

func (r *waveRepository) FindByID(ctx context.Context, id uint) (*domain.Wave, error) {
	var p persistence.Wave
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.WaveToDomain(&p), nil
}

func (r *waveRepository) FindByWaveNo(ctx context.Context, waveNo string) (*domain.Wave, error) {
	var p persistence.Wave
	if err := r.db.WithContext(ctx).Where("wave_no = ?", waveNo).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.WaveToDomain(&p), nil
}

func (r *waveRepository) List(ctx context.Context) ([]domain.Wave, error) {
	var ps []persistence.Wave
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Wave, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveToDomain(&p)
	}
	return result, nil
}

func (r *waveRepository) ListPaginated(ctx context.Context, offset, limit int) ([]domain.Wave, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&persistence.Wave{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ps []persistence.Wave
	if err := r.db.WithContext(ctx).Order("id").Offset(offset).Limit(limit).Find(&ps).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domain.Wave, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveToDomain(&p)
	}
	return result, total, nil
}

func (r *waveRepository) AddParticipant(ctx context.Context, snap *domain.WaveParticipantSnapshot) error {
	p := persistence.WaveParticipantSnapshotFromDomain(snap)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*snap = *persistence.WaveParticipantSnapshotToDomain(p)
	return nil
}

func (r *waveRepository) ListParticipantsByWave(ctx context.Context, waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	var ps []persistence.WaveParticipantSnapshot
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveParticipantSnapshot, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveParticipantSnapshotToDomain(&p)
	}
	return result, nil
}

func (r *waveRepository) ListParticipantsByProfile(ctx context.Context, profileID uint) ([]domain.WaveParticipantSnapshot, error) {
	var ps []persistence.WaveParticipantSnapshot
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveParticipantSnapshot, len(ps))
	for i, p := range ps {
		result[i] = *persistence.WaveParticipantSnapshotToDomain(&p)
	}
	return result, nil
}

func (r *waveRepository) UpdateLifecycle(ctx context.Context, waveID uint, stage string, progressSnapshot string) error {
	return r.db.WithContext(ctx).Model(&persistence.Wave{}).Where("id = ?", waveID).Updates(map[string]interface{}{
		"lifecycle_stage":   stage,
		"progress_snapshot": progressSnapshot,
	}).Error
}

func (r *waveRepository) UpdateParticipantProfileID(ctx context.Context, oldProfileID, newProfileID uint) (int64, error) {
	res := r.db.WithContext(ctx).Model(&persistence.WaveParticipantSnapshot{}).
		Where("customer_profile_id = ?", oldProfileID).
		Update("customer_profile_id", newProfileID)
	return res.RowsAffected, res.Error
}

func (r *waveRepository) DeleteParticipantsByWave(ctx context.Context, waveID uint) error {
	// WaveParticipantSnapshot has no DeletedAt (no soft-delete); this is a hard delete.
	return r.db.WithContext(ctx).Where("wave_id = ?", waveID).Delete(&persistence.WaveParticipantSnapshot{}).Error
}

func (r *waveRepository) CountByDatePrefix(ctx context.Context, prefix string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&persistence.Wave{}).Where("wave_no LIKE ?", prefix+"%").Count(&count).Error
	return int(count), err
}
