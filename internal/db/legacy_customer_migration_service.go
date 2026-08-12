package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const defaultLegacyCustomerMigrationBatchSize = 200

func legacyCustomerDataMigrations() []batchedDataMigration {
	return []batchedDataMigration{
		{
			Version:   1,
			Name:      "legacy_customer_import_v1",
			Signature: "discover members/member_nicknames; map exact platform+platform_uid; batch cursors; quarantine dirty rows; nickname event key old id+created_at",
			Up: func(database *gorm.DB) (string, uint64, error) {
				result, err := RunLegacyCustomerMigration(context.Background(), database, defaultLegacyCustomerMigrationBatchSize)
				if err != nil {
					return "", 0, err
				}
				checkpoint := "complete"
				if !result.LegacySchemaFound {
					checkpoint = "legacy_schema_not_present"
				}
				return checkpoint, uint64(result.MembersRead + result.NicknamesRead), nil
			},
		},
		{
			Version:   2,
			Name:      "legacy_customer_name_compatibility_v2",
			Signature: "legacy nickname episode_key becomes source event key; observed event payload uses shared CustomerNameEventPayload trusted nickname metadata",
			Up: func(database *gorm.DB) (string, uint64, error) {
				rowsProcessed, err := repairLegacyCustomerNameCompatibilityV2(context.Background(), database, defaultLegacyCustomerMigrationBatchSize)
				if err != nil {
					return "", 0, err
				}
				return "complete", rowsProcessed, nil
			},
		},
	}
}

type LegacyCustomerMigrationResult struct {
	LegacySchemaFound bool
	MembersRead       int
	NicknamesRead     int
	ProfilesCreated   int
	IdentitiesCreated int
	MappingsCreated   int
	ObservationsAdded int
	NameEventsAdded   int
	RowsQuarantined   int
}

type LegacyCustomerMigrationService struct {
	reader     LegacyCustomerReader
	repository LegacyCustomerRepository
	batchSize  int
}

func NewLegacyCustomerMigrationService(database *gorm.DB, batchSize int) *LegacyCustomerMigrationService {
	if batchSize <= 0 {
		batchSize = defaultLegacyCustomerMigrationBatchSize
	}
	return &LegacyCustomerMigrationService{
		reader:     newSQLiteLegacyCustomerReader(database),
		repository: newSQLiteLegacyCustomerRepository(database),
		batchSize:  batchSize,
	}
}

func RunLegacyCustomerMigration(
	ctx context.Context,
	database *gorm.DB,
	batchSize int,
) (*LegacyCustomerMigrationResult, error) {
	return NewLegacyCustomerMigrationService(database, batchSize).Run(ctx)
}

func (s *LegacyCustomerMigrationService) Run(ctx context.Context) (*LegacyCustomerMigrationResult, error) {
	if s == nil || s.reader == nil || s.repository == nil {
		return nil, fmt.Errorf("run legacy customer migration: service is not initialized")
	}
	schema, err := s.reader.Discover(ctx)
	if err != nil {
		return nil, err
	}
	result := &LegacyCustomerMigrationResult{LegacySchemaFound: schema.MembersPresent}
	if !schema.MembersPresent {
		return result, nil
	}
	if err := s.migrateMembers(ctx, schema, result); err != nil {
		return nil, err
	}
	if schema.NicknamesPresent {
		if err := s.migrateNicknames(ctx, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *LegacyCustomerMigrationService) migrateMembers(
	ctx context.Context,
	schema *LegacyCustomerSchema,
	result *LegacyCustomerMigrationResult,
) error {
	cursor, err := s.repository.GetCursor(ctx, "members")
	if err != nil {
		return err
	}
	if cursor.Status == "complete" {
		return nil
	}
	for {
		rows, err := s.reader.ReadMembersAfter(ctx, cursor.LastLegacyID, s.batchSize, schema)
		if err != nil {
			return err
		}
		batchComplete := len(rows) < s.batchSize
		err = s.repository.Transaction(ctx, func(repository LegacyCustomerRepository) error {
			for _, row := range rows {
				if err := migrateLegacyMemberRow(ctx, repository, row, result); err != nil {
					return err
				}
				cursor.LastLegacyID = row.ID
				result.MembersRead++
			}
			cursor.Status = "running"
			if batchComplete {
				cursor.Status = "complete"
			}
			return repository.SaveCursor(ctx, cursor)
		})
		if err != nil {
			return fmt.Errorf("migrate legacy members batch after %d: %w", cursor.LastLegacyID, err)
		}
		if batchComplete {
			return nil
		}
	}
}

func migrateLegacyMemberRow(
	ctx context.Context,
	repository LegacyCustomerRepository,
	row LegacyMemberRow,
	result *LegacyCustomerMigrationResult,
) error {
	platform := row.Platform
	platformUID := row.PlatformUID
	rawPayload := marshalLegacyRow(row)
	if row.ID <= 0 || strings.TrimSpace(platform) == "" || strings.TrimSpace(platformUID) == "" {
		result.RowsQuarantined++
		return repository.UpsertQuarantine(ctx, &LegacyCustomerQuarantine{
			Stream: "members", LegacyRowID: row.ID, EventKey: fmt.Sprintf("legacy:members:%d", row.ID),
			Reason: "invalid_identity", RawPayload: rawPayload,
		})
	}

	mapping, err := repository.FindMap(ctx, row.ID)
	if err == nil {
		if mapping.LegacyPlatform != platform || mapping.LegacyPlatformUID != platformUID {
			return fmt.Errorf("legacy customer map %d identity changed from %s/%s to %s/%s",
				row.ID, mapping.LegacyPlatform, mapping.LegacyPlatformUID, platform, platformUID)
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find legacy customer map %d: %w", row.ID, err)
	}

	profileIDs, err := repository.FindIdentityProfileIDs(ctx, platform, platformUID)
	if err != nil {
		return err
	}
	if len(profileIDs) > 1 {
		result.RowsQuarantined++
		return repository.UpsertQuarantine(ctx, &LegacyCustomerQuarantine{
			Stream: "members", LegacyRowID: row.ID, EventKey: fmt.Sprintf("legacy:members:%d", row.ID),
			Reason: "ambiguous_existing_identity", RawPayload: rawPayload,
		})
	}

	var profileID uint
	if len(profileIDs) == 1 {
		profileID = profileIDs[0]
	} else {
		displayName := strings.TrimSpace(row.Nickname)
		if displayName == "" {
			displayName = platformUID
		}
		profileID, err = repository.CreateProfileAndIdentity(ctx, platform, platformUID, displayName)
		if err != nil {
			return err
		}
		result.ProfilesCreated++
		result.IdentitiesCreated++
	}

	mapping = &LegacyCustomerMap{
		LegacyMemberID: row.ID, LegacyPlatform: platform, LegacyPlatformUID: platformUID,
		CustomerProfileID: profileID,
	}
	if err := repository.CreateMap(ctx, mapping); err != nil {
		return fmt.Errorf("create legacy customer map %d: %w", row.ID, err)
	}
	result.MappingsCreated++
	return nil
}

func (s *LegacyCustomerMigrationService) migrateNicknames(
	ctx context.Context,
	result *LegacyCustomerMigrationResult,
) error {
	cursor, err := s.repository.GetCursor(ctx, "member_nicknames")
	if err != nil {
		return err
	}
	if cursor.Status == "complete" {
		return nil
	}
	for {
		rows, err := s.reader.ReadNicknamesAfter(ctx, cursor.LastLegacyID, s.batchSize)
		if err != nil {
			return err
		}
		batchComplete := len(rows) < s.batchSize
		err = s.repository.Transaction(ctx, func(repository LegacyCustomerRepository) error {
			for _, row := range rows {
				if err := migrateLegacyNicknameRow(ctx, repository, row, result); err != nil {
					return err
				}
				cursor.LastLegacyID = row.ID
				result.NicknamesRead++
			}
			cursor.Status = "running"
			if batchComplete {
				cursor.Status = "complete"
			}
			return repository.SaveCursor(ctx, cursor)
		})
		if err != nil {
			return fmt.Errorf("migrate legacy member_nicknames batch after %d: %w", cursor.LastLegacyID, err)
		}
		if batchComplete {
			return nil
		}
	}
}

func migrateLegacyNicknameRow(
	ctx context.Context,
	repository LegacyCustomerRepository,
	row LegacyMemberNicknameRow,
	result *LegacyCustomerMigrationResult,
) error {
	createdAtRaw := row.CreatedAtRaw
	eventKey := fmt.Sprintf("legacy:member_nicknames:%d:%s", row.ID, createdAtRaw)
	rawPayload := marshalLegacyRow(row)
	nickname := row.Nickname
	observedAt, timestampErr := parseLegacyTimestamp(createdAtRaw)
	if row.ID <= 0 || row.MemberID <= 0 || strings.TrimSpace(nickname) == "" || timestampErr != nil {
		reason := "invalid_nickname"
		if timestampErr != nil {
			reason = "invalid_created_at"
		}
		result.RowsQuarantined++
		return repository.UpsertQuarantine(ctx, &LegacyCustomerQuarantine{
			Stream: "member_nicknames", LegacyRowID: row.ID, EventKey: eventKey,
			Reason: reason, RawPayload: rawPayload,
		})
	}

	mapping, err := repository.FindMap(ctx, row.MemberID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result.RowsQuarantined++
		return repository.UpsertQuarantine(ctx, &LegacyCustomerQuarantine{
			Stream: "member_nicknames", LegacyRowID: row.ID, EventKey: eventKey,
			Reason: "legacy_member_not_mapped", RawPayload: rawPayload,
		})
	}
	if err != nil {
		return fmt.Errorf("find legacy customer map %d for nickname %d: %w", row.MemberID, row.ID, err)
	}

	observationCreated, eventCreated, err := repository.InsertNicknameObservation(ctx, LegacyNicknameObservationInput{
		CustomerProfileID: mapping.CustomerProfileID,
		LegacyMemberID:    row.MemberID,
		LegacyNicknameID:  row.ID,
		Nickname:          nickname,
		NormalizedName:    strings.ToLower(strings.TrimSpace(nickname)),
		ObservedAt:        observedAt,
		EventKey:          eventKey,
		EpisodeKey:        eventKey,
		ExtraData:         rawPayload,
	})
	if err != nil {
		return err
	}
	if observationCreated {
		result.ObservationsAdded++
	}
	if eventCreated {
		result.NameEventsAdded++
	}
	return nil
}

func marshalLegacyRow(row any) string {
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Sprintf(`{"marshalError":%q}`, err.Error())
	}
	return string(payload)
}

func parseLegacyTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	if unixValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		if len(value) >= 13 {
			return time.UnixMilli(unixValue).UTC(), nil
		}
		return time.Unix(unixValue, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid legacy timestamp %q", value)
}
