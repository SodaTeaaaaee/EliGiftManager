package infra

import (
	"time"

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

func (r *channelSyncRepository) SaveJob(job *domain.ChannelSyncJob) error {
	// Load existing row to avoid overwriting fields not carried by the domain object.
	var existing persistence.ChannelSyncJob
	if err := r.db.First(&existing, job.ID).Error; err != nil {
		return err
	}

	// Only patch runtime action fields — never overwrite CreatedAt, DeletedAt, or
	// future columns that the domain object doesn't carry.
	existing.Status = persistence.ChannelSyncJobStatus(job.Status)
	existing.RequestPayload = job.RequestPayload
	existing.ResponsePayload = job.ResponsePayload
	existing.ErrorMessage = job.ErrorMessage
	existing.BasisHistoryNodeID = job.BasisHistoryNodeID
	existing.BasisProjectionHash = job.BasisProjectionHash
	existing.BasisPayloadSnapshot = job.BasisPayloadSnapshot

	if job.StartedAt != "" {
		t, _ := time.Parse(time.RFC3339, job.StartedAt)
		existing.StartedAt = &t
	}
	if job.FinishedAt != "" {
		t, _ := time.Parse(time.RFC3339, job.FinishedAt)
		existing.FinishedAt = &t
	}

	if err := r.db.Save(&existing).Error; err != nil {
		return err
	}
	*job = *persistence.FromPersistenceChannelSyncJob(&existing)
	return nil
}

func (r *channelSyncRepository) SaveItem(item *domain.ChannelSyncItem) error {
	var existing persistence.ChannelSyncItem
	if err := r.db.First(&existing, item.ID).Error; err != nil {
		return err
	}

	existing.Status = persistence.ChannelSyncItemStatus(item.Status)
	existing.ErrorMessage = item.ErrorMessage
	existing.ExternalDocumentNo = item.ExternalDocumentNo
	existing.ExternalLineNo = item.ExternalLineNo
	existing.TrackingNo = item.TrackingNo
	existing.CarrierCode = item.CarrierCode

	if err := r.db.Save(&existing).Error; err != nil {
		return err
	}
	*item = *persistence.FromPersistenceChannelSyncItem(&existing)
	return nil
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

func (r *channelSyncRepository) AtomicCreateChannelSync(job *domain.ChannelSyncJob, items []*domain.ChannelSyncItem, pin *domain.BasisPinParam) error {
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
		if pin != nil && pin.HistoryNodeID != 0 {
			pPin := &persistence.HistoryPin{
				HistoryNodeID: pin.HistoryNodeID,
				PinKind:       pin.PinKind,
				RefType:       pin.RefType,
				RefID:         job.ID,
			}
			if err := tx.Create(pPin).Error; err != nil {
				return err
			}
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

func (r *channelSyncRepository) CountJobsByProfileID(profileID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&persistence.ChannelSyncJob{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
