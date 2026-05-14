package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type channelSyncRepository struct {
	db *gorm.DB
}

func NewChannelSyncRepository(db *gorm.DB) domain.ChannelSyncRepository {
	return &channelSyncRepository{db: db}
}

func (r *channelSyncRepository) CreateJob(job *domain.ChannelSyncJob) error {
	p := persistence.ToPersistenceChannelSyncJob(job)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*job = *persistence.FromPersistenceChannelSyncJob(p)
	return nil
}

func (r *channelSyncRepository) FindJobByID(id uint) (*domain.ChannelSyncJob, error) {
	var p persistence.ChannelSyncJob
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceChannelSyncJob(&p), nil
}

func (r *channelSyncRepository) ListJobsByWave(waveID uint) ([]domain.ChannelSyncJob, error) {
	var ps []persistence.ChannelSyncJob
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelSyncJob, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceChannelSyncJob(&p)
	}
	return result, nil
}

func (r *channelSyncRepository) CreateItem(item *domain.ChannelSyncItem) error {
	p := persistence.ToPersistenceChannelSyncItem(item)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*item = *persistence.FromPersistenceChannelSyncItem(p)
	return nil
}

func (r *channelSyncRepository) AtomicCreateChannelSync(job *domain.ChannelSyncJob, items []*domain.ChannelSyncItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		pJob := persistence.ToPersistenceChannelSyncJob(job)
		if err := tx.Create(pJob).Error; err != nil {
			return err
		}
		*job = *persistence.FromPersistenceChannelSyncJob(pJob)
		for _, item := range items {
			item.ChannelSyncJobID = job.ID
			pItem := persistence.ToPersistenceChannelSyncItem(item)
			if err := tx.Create(pItem).Error; err != nil {
				return err
			}
			*item = *persistence.FromPersistenceChannelSyncItem(pItem)
		}
		return nil
	})
}

func (r *channelSyncRepository) ListItemsByJob(jobID uint) ([]domain.ChannelSyncItem, error) {
	var ps []persistence.ChannelSyncItem
	if err := r.db.Where("channel_sync_job_id = ?", jobID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelSyncItem, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceChannelSyncItem(&p)
	}
	return result, nil
}
