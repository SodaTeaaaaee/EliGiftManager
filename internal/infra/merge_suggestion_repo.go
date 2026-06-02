package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type mergeSuggestionRepository struct {
	db *gorm.DB
}

func NewMergeSuggestionRepository(db *gorm.DB) domain.MergeSuggestionRepository {
	return &mergeSuggestionRepository{db: db}
}

func (r *mergeSuggestionRepository) ListPending() ([]domain.MergeSuggestion, error) {
	var rows []persistence.MergeSuggestion
	if err := r.db.Where("status = ?", "pending").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergeSuggestion, len(rows))
	for i, row := range rows {
		result[i] = domain.MergeSuggestion{
			ID:              row.ID,
			SourceProfileID: row.SourceProfileID,
			TargetProfileID: row.TargetProfileID,
			Reason:          row.Reason,
			Status:          row.Status,
		}
	}
	return result, nil
}

func (r *mergeSuggestionRepository) Dismiss(id uint) error {
	return r.db.Model(&persistence.MergeSuggestion{}).Where("id = ?", id).Update("status", "dismissed").Error
}

func (r *mergeSuggestionRepository) CountBySourceAndTarget(sourceID, targetID uint) (int64, error) {
	var count int64
	err := r.db.Model(&persistence.MergeSuggestion{}).
		Where("source_profile_id = ? AND target_profile_id = ?", sourceID, targetID).
		Count(&count).Error
	return count, err
}

func (r *mergeSuggestionRepository) Create(suggestion *domain.MergeSuggestion) error {
	p := persistence.MergeSuggestion{
		SourceProfileID: suggestion.SourceProfileID,
		TargetProfileID: suggestion.TargetProfileID,
		Reason:          suggestion.Reason,
		Status:          suggestion.Status,
	}
	if err := r.db.Create(&p).Error; err != nil {
		return err
	}
	suggestion.ID = p.ID
	return nil
}

func (r *mergeSuggestionRepository) FindEmailDuplicates() ([]domain.DuplicateGroup, error) {
	type emailGroup struct {
		IdentityValue string
		ProfileIDs    string
	}
	var rows []emailGroup
	if err := r.db.Raw(`
		SELECT identity_value, GROUP_CONCAT(customer_profile_id) as profile_ids
		FROM customer_identities
		WHERE identity_type = 'email' AND deleted_at IS NULL
		GROUP BY identity_value
		HAVING COUNT(DISTINCT customer_profile_id) > 1
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DuplicateGroup, len(rows))
	for i, row := range rows {
		result[i] = domain.DuplicateGroup{
			Key:        row.IdentityValue,
			ProfileIDs: row.ProfileIDs,
		}
	}
	return result, nil
}

func (r *mergeSuggestionRepository) FindPhoneDuplicates() ([]domain.DuplicateGroup, error) {
	type phoneGroup struct {
		Phone      string
		ProfileIDs string
	}
	var rows []phoneGroup
	if err := r.db.Raw(`
		SELECT phone, GROUP_CONCAT(customer_profile_id) as profile_ids
		FROM customer_addresses
		WHERE phone != '' AND deleted_at IS NULL
		GROUP BY phone
		HAVING COUNT(DISTINCT customer_profile_id) > 1
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.DuplicateGroup, len(rows))
	for i, row := range rows {
		result[i] = domain.DuplicateGroup{
			Key:        row.Phone,
			ProfileIDs: row.ProfileIDs,
		}
	}
	return result, nil
}
