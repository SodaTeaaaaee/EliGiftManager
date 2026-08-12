package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type LegacyCustomerMap struct {
	ID                uint `gorm:"primaryKey"`
	LegacyMemberID    int64
	LegacyPlatform    string
	LegacyPlatformUID string
	CustomerProfileID uint
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (LegacyCustomerMap) TableName() string { return "legacy_customer_maps" }

type LegacyCustomerCursor struct {
	Stream       string `gorm:"primaryKey"`
	LastLegacyID int64
	Status       string
	UpdatedAt    time.Time
}

func (LegacyCustomerCursor) TableName() string { return "legacy_customer_migration_cursors" }

type LegacyCustomerQuarantine struct {
	ID          uint `gorm:"primaryKey"`
	Stream      string
	LegacyRowID int64
	EventKey    string
	Reason      string
	RawPayload  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (LegacyCustomerQuarantine) TableName() string {
	return "legacy_customer_migration_quarantines"
}

type LegacyCustomerRepository interface {
	Transaction(ctx context.Context, fn func(LegacyCustomerRepository) error) error
	GetCursor(ctx context.Context, stream string) (*LegacyCustomerCursor, error)
	SaveCursor(ctx context.Context, cursor *LegacyCustomerCursor) error
	FindMap(ctx context.Context, legacyMemberID int64) (*LegacyCustomerMap, error)
	CreateMap(ctx context.Context, mapping *LegacyCustomerMap) error
	FindIdentityProfileIDs(ctx context.Context, platform, platformUID string) ([]uint, error)
	CreateProfileAndIdentity(ctx context.Context, platform, platformUID, displayName string) (uint, error)
	UpsertQuarantine(ctx context.Context, quarantine *LegacyCustomerQuarantine) error
	InsertNicknameObservation(ctx context.Context, input LegacyNicknameObservationInput) (observationCreated, eventCreated bool, err error)
	UpdateObservationEpisodeKey(ctx context.Context, observationID int64, episodeKey string) error
	UpdateNameEventCompatibility(ctx context.Context, eventID int64, payload string) error
}

type LegacyNicknameObservationInput struct {
	CustomerProfileID uint
	LegacyMemberID    int64
	LegacyNicknameID  int64
	Nickname          string
	NormalizedName    string
	ObservedAt        time.Time
	EventKey          string
	EpisodeKey        string
	ExtraData         string
}

type sqliteLegacyCustomerRepository struct{ db *gorm.DB }

func newSQLiteLegacyCustomerRepository(db *gorm.DB) LegacyCustomerRepository {
	return &sqliteLegacyCustomerRepository{db: db}
}

func (r *sqliteLegacyCustomerRepository) Transaction(ctx context.Context, fn func(LegacyCustomerRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&sqliteLegacyCustomerRepository{db: tx})
	})
}

func (r *sqliteLegacyCustomerRepository) GetCursor(ctx context.Context, stream string) (*LegacyCustomerCursor, error) {
	var cursor LegacyCustomerCursor
	err := r.db.WithContext(ctx).Where("stream = ?", stream).Take(&cursor).Error
	if err == nil {
		return &cursor, nil
	}
	if err == gorm.ErrRecordNotFound {
		return &LegacyCustomerCursor{Stream: stream, Status: "pending"}, nil
	}
	return nil, fmt.Errorf("read legacy customer cursor %q: %w", stream, err)
}

func (r *sqliteLegacyCustomerRepository) SaveCursor(ctx context.Context, cursor *LegacyCustomerCursor) error {
	if cursor == nil || cursor.Stream == "" {
		return fmt.Errorf("save legacy customer cursor: stream is required")
	}
	now := time.Now().UTC()
	cursor.UpdatedAt = now
	return r.db.WithContext(ctx).Exec(`INSERT INTO legacy_customer_migration_cursors
(stream, last_legacy_id, status, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(stream) DO UPDATE SET
last_legacy_id = excluded.last_legacy_id, status = excluded.status, updated_at = excluded.updated_at`,
		cursor.Stream, cursor.LastLegacyID, cursor.Status, now).Error
}

func (r *sqliteLegacyCustomerRepository) FindMap(ctx context.Context, legacyMemberID int64) (*LegacyCustomerMap, error) {
	var mapping LegacyCustomerMap
	err := r.db.WithContext(ctx).Where("legacy_member_id = ?", legacyMemberID).Take(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (r *sqliteLegacyCustomerRepository) CreateMap(ctx context.Context, mapping *LegacyCustomerMap) error {
	if mapping == nil {
		return fmt.Errorf("create legacy customer map: mapping is required")
	}
	return r.db.WithContext(ctx).Create(mapping).Error
}

func (r *sqliteLegacyCustomerRepository) FindIdentityProfileIDs(
	ctx context.Context,
	platform string,
	platformUID string,
) ([]uint, error) {
	var profileIDs []uint
	err := r.db.WithContext(ctx).Table("customer_identities").
		Distinct("customer_profile_id").
		Where("identity_platform = ? AND identity_value = ? AND deleted_at IS NULL", platform, platformUID).
		Order("customer_profile_id").
		Pluck("customer_profile_id", &profileIDs).Error
	if err != nil {
		return nil, fmt.Errorf("find legacy identity %s/%s: %w", platform, platformUID, err)
	}
	return profileIDs, nil
}

func (r *sqliteLegacyCustomerRepository) CreateProfileAndIdentity(
	ctx context.Context,
	platform string,
	platformUID string,
	displayName string,
) (uint, error) {
	now := time.Now().UTC()
	profile := &persistence.CustomerProfile{
		DisplayName:     displayName,
		ProfileType:     persistence.ProfileTypeMember,
		Status:          "active",
		RowVersion:      1,
		DisplayNameMode: "auto",
	}
	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		return 0, fmt.Errorf("create profile for legacy identity %s/%s: %w", platform, platformUID, err)
	}
	identity := &persistence.CustomerIdentity{
		CustomerProfileID:    profile.ID,
		IdentityPlatform:     platform,
		IdentityValue:        platformUID,
		IdentityType:         persistence.IdentityTypePlatformUID,
		Namespace:            platform,
		NormalizedValue:      platformUID,
		NormalizationVersion: "legacy-v1",
		Authority:            "legacy_members",
		VerificationStatus:   "legacy_imported",
		ResolutionStatus:     "resolved",
		FirstSeenAt:          &now,
		LastSeenAt:           &now,
		IsPrimary:            true,
	}
	if err := r.db.WithContext(ctx).Create(identity).Error; err != nil {
		return 0, fmt.Errorf("create legacy identity %s/%s: %w", platform, platformUID, err)
	}
	return profile.ID, nil
}

func (r *sqliteLegacyCustomerRepository) UpsertQuarantine(
	ctx context.Context,
	quarantine *LegacyCustomerQuarantine,
) error {
	if quarantine == nil || quarantine.Stream == "" {
		return fmt.Errorf("upsert legacy customer quarantine: stream is required")
	}
	now := time.Now().UTC()
	if quarantine.CreatedAt.IsZero() {
		quarantine.CreatedAt = now
	}
	quarantine.UpdatedAt = now
	return r.db.WithContext(ctx).Exec(`INSERT INTO legacy_customer_migration_quarantines
(stream, legacy_row_id, event_key, reason, raw_payload, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(stream, legacy_row_id) DO UPDATE SET
event_key = excluded.event_key, reason = excluded.reason,
raw_payload = excluded.raw_payload, updated_at = excluded.updated_at`,
		quarantine.Stream, quarantine.LegacyRowID, quarantine.EventKey, quarantine.Reason,
		quarantine.RawPayload, quarantine.CreatedAt, quarantine.UpdatedAt).Error
}

func (r *sqliteLegacyCustomerRepository) InsertNicknameObservation(
	ctx context.Context,
	input LegacyNicknameObservationInput,
) (bool, bool, error) {
	result := r.db.WithContext(ctx).Exec(`INSERT OR IGNORE INTO customer_name_observations
(created_at, updated_at, customer_profile_id, name, normalized_name, name_kind,
 authority, trust_score, observed_at, first_seen_at, last_seen_at, is_pinned,
 is_active, extra_data, source_event_key, episode_key, observation_count)
VALUES (?, ?, ?, ?, ?, 'trusted_nickname', 'legacy_member_nicknames', 0.8, ?, ?, ?, false, true, ?, ?, ?, 1)`,
		input.ObservedAt, input.ObservedAt, input.CustomerProfileID, input.Nickname, input.NormalizedName,
		input.ObservedAt, input.ObservedAt, input.ObservedAt, input.ExtraData, input.EventKey, input.EpisodeKey)
	if result.Error != nil {
		return false, false, fmt.Errorf("insert legacy nickname observation %q: %w", input.EventKey, result.Error)
	}
	observationCreated := result.RowsAffected == 1
	var observationID uint
	if err := r.db.WithContext(ctx).Table("customer_name_observations").
		Select("id").Where("source_event_key = ?", input.EventKey).Scan(&observationID).Error; err != nil {
		return false, false, fmt.Errorf("find legacy nickname observation %q: %w", input.EventKey, err)
	}
	if observationID == 0 {
		return false, false, fmt.Errorf("legacy nickname observation %q was not persisted", input.EventKey)
	}
	eventPayload, err := json.Marshal(domain.CustomerNameEventPayload{
		NameKind: domain.CustomerNameKindTrustedNickname, Authority: "legacy_member_nicknames",
		TrustScore: 0.8, ExtraData: input.ExtraData,
	})
	if err != nil {
		return false, false, fmt.Errorf("encode legacy nickname event %q: %w", input.EventKey, err)
	}
	eventResult := r.db.WithContext(ctx).Exec(`INSERT OR IGNORE INTO customer_name_events
(customer_profile_id, observation_id, event_kind, previous_name, new_name,
 reason_code, actor_ref, payload, created_at, event_key)
VALUES (?, ?, 'observed', '', ?, 'trusted_nickname', 'legacy_member_nicknames', ?, ?, ?)`,
		input.CustomerProfileID, observationID, input.Nickname, string(eventPayload), input.ObservedAt, input.EventKey)
	if eventResult.Error != nil {
		return false, false, fmt.Errorf("insert legacy nickname event %q: %w", input.EventKey, eventResult.Error)
	}
	return observationCreated, eventResult.RowsAffected == 1, nil
}

func (r *sqliteLegacyCustomerRepository) UpdateObservationEpisodeKey(
	ctx context.Context,
	observationID int64,
	episodeKey string,
) error {
	return r.db.WithContext(ctx).Table("customer_name_observations").Where("id = ?", observationID).
		Update("episode_key", episodeKey).Error
}

func (r *sqliteLegacyCustomerRepository) UpdateNameEventCompatibility(
	ctx context.Context,
	eventID int64,
	payload string,
) error {
	return r.db.WithContext(ctx).Table("customer_name_events").Where("id = ?", eventID).Updates(map[string]any{
		"payload": payload, "reason_code": domain.CustomerNameKindTrustedNickname,
		"actor_ref": "legacy_member_nicknames",
	}).Error
}
