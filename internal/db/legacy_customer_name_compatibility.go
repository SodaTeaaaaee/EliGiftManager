package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

const (
	legacyObservationCompatibilityStream = "legacy_name_observation_episode_v2"
	legacyEventCompatibilityStream       = "legacy_name_event_payload_v2"
)

func repairLegacyCustomerNameCompatibilityV2(
	ctx context.Context,
	database *gorm.DB,
	batchSize int,
) (uint64, error) {
	if batchSize <= 0 {
		batchSize = defaultLegacyCustomerMigrationBatchSize
	}
	repository := newSQLiteLegacyCustomerRepository(database)
	observationRows, err := repairLegacyObservationEpisodes(ctx, database, repository, batchSize)
	if err != nil {
		return 0, err
	}
	eventRows, err := repairLegacyNameEventPayloads(ctx, database, repository, batchSize)
	if err != nil {
		return 0, err
	}
	return observationRows + eventRows, nil
}

func repairLegacyObservationEpisodes(
	ctx context.Context,
	database *gorm.DB,
	repository LegacyCustomerRepository,
	batchSize int,
) (uint64, error) {
	cursor, err := repository.GetCursor(ctx, legacyObservationCompatibilityStream)
	if err != nil {
		return 0, err
	}
	if cursor.Status == "complete" {
		return 0, nil
	}
	type observationRow struct {
		ID             int64
		SourceEventKey string
	}
	var processed uint64
	for {
		var rows []observationRow
		err := database.WithContext(ctx).Table("customer_name_observations").
			Select("id, source_event_key").
			Where("id > ? AND source_event_key LIKE ?", cursor.LastLegacyID, "legacy:member_nicknames:%").
			Order("id").Limit(batchSize).Scan(&rows).Error
		if err != nil {
			return 0, fmt.Errorf("read legacy nickname observations for v2 compatibility: %w", err)
		}
		batchComplete := len(rows) < batchSize
		err = repository.Transaction(ctx, func(txRepository LegacyCustomerRepository) error {
			for _, row := range rows {
				if err := txRepository.UpdateObservationEpisodeKey(ctx, row.ID, row.SourceEventKey); err != nil {
					return fmt.Errorf("repair legacy nickname observation %d episode key: %w", row.ID, err)
				}
				cursor.LastLegacyID = row.ID
				processed++
			}
			cursor.Status = "running"
			if batchComplete {
				cursor.Status = "complete"
			}
			return txRepository.SaveCursor(ctx, cursor)
		})
		if err != nil {
			return 0, err
		}
		if batchComplete {
			return processed, nil
		}
	}
}

func repairLegacyNameEventPayloads(
	ctx context.Context,
	database *gorm.DB,
	repository LegacyCustomerRepository,
	batchSize int,
) (uint64, error) {
	cursor, err := repository.GetCursor(ctx, legacyEventCompatibilityStream)
	if err != nil {
		return 0, err
	}
	if cursor.Status == "complete" {
		return 0, nil
	}
	type eventRow struct {
		ID      int64
		Payload string
	}
	var processed uint64
	for {
		var rows []eventRow
		err := database.WithContext(ctx).Table("customer_name_events").
			Select("id, payload").
			Where("id > ? AND event_kind = ? AND event_key LIKE ?", cursor.LastLegacyID, "observed", "legacy:member_nicknames:%").
			Order("id").Limit(batchSize).Scan(&rows).Error
		if err != nil {
			return 0, fmt.Errorf("read legacy nickname events for v2 compatibility: %w", err)
		}
		batchComplete := len(rows) < batchSize
		err = repository.Transaction(ctx, func(txRepository LegacyCustomerRepository) error {
			for _, row := range rows {
				metadata := domain.CustomerNameEventPayload{}
				if err := json.Unmarshal([]byte(row.Payload), &metadata); err != nil {
					return fmt.Errorf("decode legacy nickname event %d payload: %w", row.ID, err)
				}
				if metadata.NameKind == "" {
					metadata = domain.CustomerNameEventPayload{
						NameKind: domain.CustomerNameKindTrustedNickname, Authority: "legacy_member_nicknames",
						TrustScore: 0.8, ExtraData: row.Payload,
					}
				}
				payload, err := json.Marshal(metadata)
				if err != nil {
					return fmt.Errorf("encode legacy nickname event %d payload: %w", row.ID, err)
				}
				if err := txRepository.UpdateNameEventCompatibility(ctx, row.ID, string(payload)); err != nil {
					return fmt.Errorf("repair legacy nickname event %d payload: %w", row.ID, err)
				}
				cursor.LastLegacyID = row.ID
				processed++
			}
			cursor.Status = "running"
			if batchComplete {
				cursor.Status = "complete"
			}
			return txRepository.SaveCursor(ctx, cursor)
		})
		if err != nil {
			return 0, err
		}
		if batchComplete {
			return processed, nil
		}
	}
}
