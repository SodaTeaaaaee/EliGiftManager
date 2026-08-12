package infra

import (
	"context"
	"errors"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type customerResolutionRepository struct{ db *gorm.DB }

var (
	_ domain.CustomerNameObservationRepository   = (*customerResolutionRepository)(nil)
	_ domain.CustomerNameEventRepository         = (*customerNameEventRepository)(nil)
	_ domain.CustomerProfileOriginRepository     = (*customerProfileOriginRepository)(nil)
	_ domain.CustomerProfileOriginReadRepository = (*customerProfileOriginRepository)(nil)
	_ domain.MergeCandidateRepository            = (*mergeCandidateRepository)(nil)
	_ domain.MergeEvidenceRepository             = (*mergeEvidenceRepository)(nil)
	_ domain.MergePolicyRepository               = (*mergePolicyRepository)(nil)
	_ domain.MergeMovedEntityRepository          = (*mergeMovedEntityRepository)(nil)
)

func NewCustomerNameObservationRepository(db *gorm.DB) domain.CustomerNameObservationRepository {
	return &customerResolutionRepository{db: db}
}

func NewCustomerNameEventRepository(db *gorm.DB) domain.CustomerNameEventRepository {
	return &customerNameEventRepository{db: db}
}

func NewCustomerProfileOriginRepository(db *gorm.DB) domain.CustomerProfileOriginRepository {
	return &customerProfileOriginRepository{db: db}
}

func NewMergeCandidateRepository(db *gorm.DB) domain.MergeCandidateRepository {
	return &mergeCandidateRepository{db: db}
}

func NewMergeEvidenceRepository(db *gorm.DB) domain.MergeEvidenceRepository {
	return &mergeEvidenceRepository{db: db}
}

func NewMergePolicyRepository(db *gorm.DB) domain.MergePolicyRepository {
	return &mergePolicyRepository{db: db}
}

func NewMergeMovedEntityRepository(db *gorm.DB) domain.MergeMovedEntityRepository {
	return &mergeMovedEntityRepository{db: db}
}

func (r *customerResolutionRepository) Create(ctx context.Context, observation *domain.CustomerNameObservation) error {
	p := persistence.CustomerNameObservationFromDomain(observation)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*observation = *persistence.CustomerNameObservationToDomain(p)
	return nil
}

func (r *customerResolutionRepository) Update(ctx context.Context, observation *domain.CustomerNameObservation) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerNameObservation{}).Where("id = ?", observation.ID).Updates(map[string]any{
		"customer_profile_id":           observation.CustomerProfileID,
		"name":                          observation.Name,
		"normalized_name":               observation.NormalizedName,
		"source_event_key":              observation.SourceEventKey,
		"episode_key":                   observation.EpisodeKey,
		"observation_count":             observation.ObservationCount,
		"name_kind":                     observation.NameKind,
		"authority":                     observation.Authority,
		"trust_score":                   observation.TrustScore,
		"source_integration_profile_id": observation.SourceIntegrationProfileID,
		"source_document_id":            observation.SourceDocumentID,
		"source_identity_id":            observation.SourceIdentityID,
		"observed_at":                   observation.ObservedAt,
		"first_seen_at":                 observation.FirstSeenAt,
		"last_seen_at":                  observation.LastSeenAt,
		"is_pinned":                     observation.IsPinned,
		"is_active":                     observation.IsActive,
		"extra_data":                    observation.ExtraData,
	}).Error
}

func (r *customerResolutionRepository) FindByID(ctx context.Context, id uint) (*domain.CustomerNameObservation, error) {
	var row persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerNameObservationToDomain(&row), nil
}

func (r *customerResolutionRepository) FindByEpisodeKey(ctx context.Context, profileID uint, episodeKey string) (*domain.CustomerNameObservation, error) {
	var row persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ? AND episode_key = ?", profileID, episodeKey).First(&row).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerNameObservationToDomain(&row), nil
}

func (r *customerResolutionRepository) FindBySourceEventKey(ctx context.Context, sourceEventKey string) (*domain.CustomerNameObservation, error) {
	var row persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).Where("source_event_key = ?", sourceEventKey).First(&row).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerNameObservationToDomain(&row), nil
}

func (r *customerResolutionRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.CustomerNameObservation, error) {
	var rows []persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("first_seen_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerNameObservation, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerNameObservationToDomain(&rows[i])
	}
	if len(result) > 0 && r.db.Migrator().HasTable("merge_moved_entities") {
		ids := make([]uint, len(result))
		for i := range result {
			ids[i] = result[i].ID
		}
		type originRow struct {
			EntityID      uint
			FromProfileID uint
		}
		var origins []originRow
		if err := r.db.WithContext(ctx).Table("merge_moved_entities").
			Select("entity_id, from_profile_id").
			Where("entity_type = ? AND entity_id IN ?", domain.MergeEntityNameObservation, ids).
			Order("id").Scan(&origins).Error; err != nil {
			return nil, err
		}
		byID := make(map[uint]uint, len(origins))
		for _, origin := range origins {
			if _, exists := byID[origin.EntityID]; !exists {
				byID[origin.EntityID] = origin.FromProfileID
			}
		}
		for i := range result {
			if originID := byID[result[i].ID]; originID != 0 {
				result[i].OriginProfileID = originID
			}
		}
	}
	return result, nil
}

func (r *customerResolutionRepository) ListByIDs(ctx context.Context, ids []uint) ([]domain.CustomerNameObservation, error) {
	if len(ids) == 0 {
		return []domain.CustomerNameObservation{}, nil
	}
	var rows []persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerNameObservation, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerNameObservationToDomain(&rows[i])
	}
	return result, nil
}

func (r *customerResolutionRepository) DeactivateByProfile(ctx context.Context, profileID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerNameObservation{}).
		Where("customer_profile_id = ? AND is_pinned = ?", profileID, false).
		Update("is_active", false).Error
}

func (r *customerResolutionRepository) BulkUpdateProfileIDByIDs(ctx context.Context, ids []uint, profileID uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&persistence.CustomerNameObservation{}).
		Where("id IN ?", ids).Update("customer_profile_id", profileID).Error
}

type customerNameEventRepository struct{ db *gorm.DB }

func (r *customerNameEventRepository) Create(ctx context.Context, event *domain.CustomerNameEvent) error {
	p := persistence.CustomerNameEventFromDomain(event)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*event = *persistence.CustomerNameEventToDomain(p)
	return nil
}

func (r *customerNameEventRepository) CreateIfAbsent(ctx context.Context, event *domain.CustomerNameEvent) (bool, error) {
	p := persistence.CustomerNameEventFromDomain(event)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(p)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		existing, err := r.FindByEventKey(ctx, event.EventKey)
		if err != nil {
			return false, err
		}
		*event = *existing
		return false, nil
	}
	*event = *persistence.CustomerNameEventToDomain(p)
	return true, nil
}

func (r *customerNameEventRepository) FindByEventKey(ctx context.Context, eventKey string) (*domain.CustomerNameEvent, error) {
	var row persistence.CustomerNameEvent
	if err := r.db.WithContext(ctx).Where("event_key = ?", eventKey).First(&row).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerNameEventToDomain(&row), nil
}

func (r *customerNameEventRepository) UpdateObservationID(ctx context.Context, eventID uint, observationID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerNameEvent{}).
		Where("id = ?", eventID).Update("observation_id", observationID).Error
}

func (r *customerNameEventRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.CustomerNameEvent, error) {
	var rows []persistence.CustomerNameEvent
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerNameEvent, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerNameEventToDomain(&rows[i])
	}
	return result, nil
}

type customerProfileOriginRepository struct{ db *gorm.DB }

func (r *customerProfileOriginRepository) Create(ctx context.Context, origin *domain.CustomerProfileOrigin) error {
	p := persistence.CustomerProfileOriginFromDomain(origin)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*origin = *persistence.CustomerProfileOriginToDomain(p)
	return nil
}

func (r *customerProfileOriginRepository) CreateIfAbsent(ctx context.Context, origin *domain.CustomerProfileOrigin) (bool, error) {
	p := persistence.CustomerProfileOriginFromDomain(origin)
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(p)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		if origin.SourceIntegrationProfileID == nil {
			return false, errors.New("origin conflict without source integration profile")
		}
		existing, err := r.FindByExternalRef(ctx, origin.OriginKind, *origin.SourceIntegrationProfileID, origin.ExternalRef)
		if err != nil {
			return false, err
		}
		*origin = *existing
		return false, nil
	}
	*origin = *persistence.CustomerProfileOriginToDomain(p)
	return true, nil
}

func (r *customerProfileOriginRepository) FindByExternalRef(ctx context.Context, originKind string, integrationProfileID uint, externalRef string) (*domain.CustomerProfileOrigin, error) {
	var row persistence.CustomerProfileOrigin
	if err := r.db.WithContext(ctx).Where(
		"origin_kind = ? AND source_integration_profile_id = ? AND external_ref = ?",
		originKind, integrationProfileID, externalRef,
	).First(&row).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerProfileOriginToDomain(&row), nil
}

func (r *customerProfileOriginRepository) FindByID(ctx context.Context, id uint) (*domain.CustomerProfileOrigin, error) {
	var row persistence.CustomerProfileOrigin
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerProfileOriginToDomain(&row), nil
}

func (r *customerProfileOriginRepository) Update(ctx context.Context, origin *domain.CustomerProfileOrigin) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).Where("id = ?", origin.ID).Updates(map[string]any{
		"customer_profile_id":           origin.CustomerProfileID,
		"origin_kind":                   origin.OriginKind,
		"source_integration_profile_id": origin.SourceIntegrationProfileID,
		"source_document_id":            origin.SourceDocumentID,
		"external_ref":                  origin.ExternalRef,
		"is_provisional":                origin.IsProvisional,
		"first_seen_at":                 origin.FirstSeenAt,
		"last_seen_at":                  origin.LastSeenAt,
		"extra_data":                    origin.ExtraData,
	}).Error
}

func (r *customerProfileOriginRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.CustomerProfileOrigin, error) {
	var rows []persistence.CustomerProfileOrigin
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerProfileOrigin, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerProfileOriginToDomain(&rows[i])
	}
	return result, nil
}

func (r *customerProfileOriginRepository) ListForProfileRead(ctx context.Context, profileID uint) ([]domain.CustomerProfileOrigin, error) {
	query := r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).
		Where("customer_profile_id = ?", profileID)
	if r.db.Migrator().HasTable("merge_moved_entities") && r.db.Migrator().HasTable("customer_merge_records") {
		ledgerIDs := r.db.WithContext(ctx).Table("merge_moved_entities AS moved").
			Select("moved.entity_id").
			Joins("JOIN customer_merge_records AS records ON records.id = moved.merge_record_id").
			Where("moved.entity_type = ? AND moved.from_profile_id = ?", domain.MergeEntityOrigin, profileID).
			Where("records.status = ? AND records.undone_at IS NULL", domain.MergeRecordStatusCompleted)
		query = r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).
			Where("customer_profile_id = ? OR id IN (?)", profileID, ledgerIDs)
	}
	var rows []persistence.CustomerProfileOrigin
	if err := query.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerProfileOrigin, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerProfileOriginToDomain(&rows[i])
	}
	return result, nil
}

func (r *customerProfileOriginRepository) ListByIDs(ctx context.Context, ids []uint) ([]domain.CustomerProfileOrigin, error) {
	if len(ids) == 0 {
		return []domain.CustomerProfileOrigin{}, nil
	}
	var rows []persistence.CustomerProfileOrigin
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerProfileOrigin, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerProfileOriginToDomain(&rows[i])
	}
	return result, nil
}

func (r *customerProfileOriginRepository) BulkUpdateProfileIDByIDs(ctx context.Context, ids []uint, profileID uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).
		Where("id IN ?", ids).Update("customer_profile_id", profileID).Error
}

type mergeCandidateRepository struct{ db *gorm.DB }

func (r *mergeCandidateRepository) Create(ctx context.Context, candidate *domain.MergeCandidate) error {
	p := persistence.MergeCandidateFromDomain(candidate)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*candidate = *persistence.MergeCandidateToDomain(p)
	return nil
}

func (r *mergeCandidateRepository) FindByID(ctx context.Context, id uint) (*domain.MergeCandidate, error) {
	var row persistence.MergeCandidate
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.MergeCandidateToDomain(&row), nil
}

func (r *mergeCandidateRepository) ListPending(ctx context.Context) ([]domain.MergeCandidate, error) {
	var rows []persistence.MergeCandidate
	if err := r.db.WithContext(ctx).Where("status = ?", "pending").Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergeCandidate, len(rows))
	for i := range rows {
		result[i] = *persistence.MergeCandidateToDomain(&rows[i])
	}
	return result, nil
}

func (r *mergeCandidateRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).Where("id = ?", id).Update("status", status).Error
}

type mergeEvidenceRepository struct{ db *gorm.DB }

func (r *mergeEvidenceRepository) Create(ctx context.Context, evidence *domain.MergeEvidence) error {
	p := persistence.MergeEvidenceFromDomain(evidence)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*evidence = *persistence.MergeEvidenceToDomain(p)
	return nil
}

func (r *mergeEvidenceRepository) ListByCandidate(ctx context.Context, candidateID uint) ([]domain.MergeEvidence, error) {
	var rows []persistence.MergeEvidence
	if err := r.db.WithContext(ctx).Where("merge_candidate_id = ?", candidateID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergeEvidence, len(rows))
	for i := range rows {
		result[i] = *persistence.MergeEvidenceToDomain(&rows[i])
	}
	return result, nil
}

type mergePolicyRepository struct{ db *gorm.DB }

func (r *mergePolicyRepository) Create(ctx context.Context, policy *domain.MergePolicy) error {
	p := persistence.MergePolicyFromDomain(policy)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*policy = *persistence.MergePolicyToDomain(p)
	return nil
}

func (r *mergePolicyRepository) CreateRevision(ctx context.Context, revision *domain.MergePolicyRevision) error {
	p := persistence.MergePolicyRevisionFromDomain(revision)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*revision = *persistence.MergePolicyRevisionToDomain(p)
	return nil
}

func (r *mergePolicyRepository) ListActive(ctx context.Context) ([]domain.MergePolicy, error) {
	var rows []persistence.MergePolicy
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergePolicy, len(rows))
	for i := range rows {
		result[i] = *persistence.MergePolicyToDomain(&rows[i])
	}
	return result, nil
}

func (r *mergePolicyRepository) ListRevisions(ctx context.Context, policyID uint) ([]domain.MergePolicyRevision, error) {
	var rows []persistence.MergePolicyRevision
	if err := r.db.WithContext(ctx).Where("merge_policy_id = ?", policyID).Order("revision").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergePolicyRevision, len(rows))
	for i := range rows {
		result[i] = *persistence.MergePolicyRevisionToDomain(&rows[i])
	}
	return result, nil
}

type mergeMovedEntityRepository struct{ db *gorm.DB }

func (r *mergeMovedEntityRepository) Create(ctx context.Context, moved *domain.MergeMovedEntity) error {
	p := persistence.MergeMovedEntityFromDomain(moved)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*moved = *persistence.MergeMovedEntityToDomain(p)
	return nil
}

func (r *mergeMovedEntityRepository) ListByMergeRecord(ctx context.Context, mergeRecordID uint) ([]domain.MergeMovedEntity, error) {
	var rows []persistence.MergeMovedEntity
	if err := r.db.WithContext(ctx).Where("merge_record_id = ?", mergeRecordID).Order("move_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergeMovedEntity, len(rows))
	for i := range rows {
		result[i] = *persistence.MergeMovedEntityToDomain(&rows[i])
	}
	return result, nil
}
