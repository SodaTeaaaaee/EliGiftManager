package infra

import (
	"context"

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

func (r *channelSyncRepository) CreateJob(ctx context.Context, job *domain.ChannelSyncJob) error {
	p := persistence.ChannelSyncJobFromDomain(job)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*job = *persistence.ChannelSyncJobToDomain(p)
	return nil
}

func (r *channelSyncRepository) FindJobByID(ctx context.Context, id uint) (*domain.ChannelSyncJob, error) {
	var p persistence.ChannelSyncJob
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.ChannelSyncJobToDomain(&p), nil
}

func (r *channelSyncRepository) SaveJob(ctx context.Context, job *domain.ChannelSyncJob) error {
	// Load existing row to avoid overwriting fields not carried by the domain object.
	var existing persistence.ChannelSyncJob
	if err := r.db.WithContext(ctx).First(&existing, job.ID).Error; err != nil {
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

	existing.StartedAt = job.StartedAt
	existing.FinishedAt = job.FinishedAt

	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*job = *persistence.ChannelSyncJobToDomain(&existing)
	return nil
}

func (r *channelSyncRepository) SaveItem(ctx context.Context, item *domain.ChannelSyncItem) error {
	var existing persistence.ChannelSyncItem
	if err := r.db.WithContext(ctx).First(&existing, item.ID).Error; err != nil {
		return err
	}

	existing.Status = persistence.ChannelSyncItemStatus(item.Status)
	existing.ErrorMessage = item.ErrorMessage
	existing.ExternalDocumentNo = item.ExternalDocumentNo
	existing.ExternalLineNo = item.ExternalLineNo
	existing.TrackingNo = item.TrackingNo
	existing.CarrierCode = item.CarrierCode

	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*item = *persistence.ChannelSyncItemToDomain(&existing)
	return nil
}

func (r *channelSyncRepository) ListJobsByWave(ctx context.Context, waveID uint) ([]domain.ChannelSyncJob, error) {
	var ps []persistence.ChannelSyncJob
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelSyncJob, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ChannelSyncJobToDomain(&p)
	}
	return result, nil
}

func (r *channelSyncRepository) CreateItem(ctx context.Context, item *domain.ChannelSyncItem) error {
	p := persistence.ChannelSyncItemFromDomain(item)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*item = *persistence.ChannelSyncItemToDomain(p)
	return nil
}

func (r *channelSyncRepository) AtomicCreateChannelSync(ctx context.Context, job *domain.ChannelSyncJob, items []*domain.ChannelSyncItem, pin *domain.BasisPinParam) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pJob := persistence.ChannelSyncJobFromDomain(job)
		if err := tx.Create(pJob).Error; err != nil {
			return err
		}
		*job = *persistence.ChannelSyncJobToDomain(pJob)
		for _, item := range items {
			item.ChannelSyncJobID = job.ID
			pItem := persistence.ChannelSyncItemFromDomain(item)
			if err := tx.Create(pItem).Error; err != nil {
				return err
			}
			*item = *persistence.ChannelSyncItemToDomain(pItem)
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

func (r *channelSyncRepository) ListItemsByJob(ctx context.Context, jobID uint) ([]domain.ChannelSyncItem, error) {
	var ps []persistence.ChannelSyncItem
	if err := r.db.WithContext(ctx).Where("channel_sync_job_id = ?", jobID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelSyncItem, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ChannelSyncItemToDomain(&p)
	}
	return result, nil
}

func (r *channelSyncRepository) CountJobsByProfileID(ctx context.Context, profileID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.ChannelSyncJob{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
