package infra

import (
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

func (r *waveRepository) Create(wave *domain.Wave) error {
	p := persistence.ToPersistenceWave(wave)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*wave = *persistence.FromPersistenceWave(p)
	return nil
}

func (r *waveRepository) FindByID(id uint) (*domain.Wave, error) {
	var p persistence.Wave
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceWave(&p), nil
}

func (r *waveRepository) FindByWaveNo(waveNo string) (*domain.Wave, error) {
	var p persistence.Wave
	if err := r.db.Where("wave_no = ?", waveNo).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceWave(&p), nil
}

func (r *waveRepository) List() ([]domain.Wave, error) {
	var ps []persistence.Wave
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Wave, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWave(&p)
	}
	return result, nil
}

func (r *waveRepository) ListPaginated(offset, limit int) ([]domain.Wave, int64, error) {
	var total int64
	if err := r.db.Model(&persistence.Wave{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ps []persistence.Wave
	if err := r.db.Order("id").Offset(offset).Limit(limit).Find(&ps).Error; err != nil {
		return nil, 0, err
	}
	result := make([]domain.Wave, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWave(&p)
	}
	return result, total, nil
}

func (r *waveRepository) AddParticipant(snap *domain.WaveParticipantSnapshot) error {
	p := persistence.ToPersistenceWaveParticipantSnapshot(snap)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*snap = *persistence.FromPersistenceWaveParticipantSnapshot(p)
	return nil
}

func (r *waveRepository) ListParticipantsByWave(waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	var ps []persistence.WaveParticipantSnapshot
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveParticipantSnapshot, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWaveParticipantSnapshot(&p)
	}
	return result, nil
}

func (r *waveRepository) ListParticipantsByProfile(profileID uint) ([]domain.WaveParticipantSnapshot, error) {
	var ps []persistence.WaveParticipantSnapshot
	if err := r.db.Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.WaveParticipantSnapshot, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceWaveParticipantSnapshot(&p)
	}
	return result, nil
}

func (r *waveRepository) UpdateLifecycle(waveID uint, stage string, progressSnapshot string) error {
	return r.db.Model(&persistence.Wave{}).Where("id = ?", waveID).Updates(map[string]interface{}{
		"lifecycle_stage":   stage,
		"progress_snapshot": progressSnapshot,
	}).Error
}

func (r *waveRepository) UpdateParticipantProfileID(oldProfileID, newProfileID uint) (int64, error) {
	res := r.db.Model(&persistence.WaveParticipantSnapshot{}).
		Where("customer_profile_id = ?", oldProfileID).
		Update("customer_profile_id", newProfileID)
	return res.RowsAffected, res.Error
}

func (r *waveRepository) DeleteParticipantsByWave(waveID uint) error {
	// WaveParticipantSnapshot has no DeletedAt (no soft-delete); this is a hard delete.
	return r.db.Where("wave_id = ?", waveID).Delete(&persistence.WaveParticipantSnapshot{}).Error
}
