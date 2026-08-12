package db

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInitDBMigratesLegacyCustomersAndNicknamesIdempotently(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "legacy-customers.db")
	legacy := openSilentSQLite(t, dbPath)
	createLegacyCustomerTables(t, legacy)
	statements := []string{
		`INSERT INTO members (id, platform, platform_uid, nickname) VALUES (5001, 'bilibili', 'uid-1', 'Current Nick')`,
		`INSERT INTO members (id, platform, platform_uid, nickname) VALUES (5002, '', 'dirty-uid', 'Dirty')`,
		`INSERT INTO member_nicknames (id, member_id, nickname, created_at) VALUES (7001, 5001, 'First Nick', '2024-01-02T03:04:05Z')`,
		`INSERT INTO member_nicknames (id, member_id, nickname, created_at) VALUES (7002, 5001, 'Second Nick', '2024-02-03T04:05:06Z')`,
		`INSERT INTO member_nicknames (id, member_id, nickname, created_at) VALUES (7003, 5002, 'Unmapped Nick', '2024-03-04T05:06:07Z')`,
	}
	for _, statement := range statements {
		if err := legacy.Exec(statement).Error; err != nil {
			t.Fatalf("seed legacy customer data: %v", err)
		}
	}
	closeGormDB(t, legacy)

	migrated, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB legacy customer migration: %v", err)
	}

	var mapping LegacyCustomerMap
	if err := migrated.Where("legacy_member_id = ?", 5001).Take(&mapping).Error; err != nil {
		t.Fatalf("read legacy customer map: %v", err)
	}
	if mapping.CustomerProfileID == 0 || mapping.CustomerProfileID == uint(mapping.LegacyMemberID) {
		t.Fatalf("legacy MemberID was treated as ProfileID: %+v", mapping)
	}
	if mapping.LegacyPlatform != "bilibili" || mapping.LegacyPlatformUID != "uid-1" {
		t.Fatalf("legacy identity mapping changed: %+v", mapping)
	}

	var identityCount int64
	if err := migrated.Table("customer_identities").Where(
		"customer_profile_id = ? AND identity_platform = ? AND identity_value = ?",
		mapping.CustomerProfileID, "bilibili", "uid-1",
	).Count(&identityCount).Error; err != nil {
		t.Fatalf("count migrated identity: %v", err)
	}
	if identityCount != 1 {
		t.Fatalf("migrated identity count = %d, want 1", identityCount)
	}

	assertTableCount(t, migrated, "customer_name_observations", 2)
	assertTableCount(t, migrated, "customer_name_events", 2)
	assertTableCount(t, migrated, "legacy_customer_migration_quarantines", 2)
	var eventKeys []string
	if err := migrated.Table("customer_name_events").Order("id").Pluck("event_key", &eventKeys).Error; err != nil {
		t.Fatalf("list migrated nickname event keys: %v", err)
	}
	wantKeys := []string{
		"legacy:member_nicknames:7001:2024-01-02T03:04:05Z",
		"legacy:member_nicknames:7002:2024-02-03T04:05:06Z",
	}
	if len(eventKeys) != len(wantKeys) || eventKeys[0] != wantKeys[0] || eventKeys[1] != wantKeys[1] {
		t.Fatalf("nickname event keys lost legacy row ID/CreatedAt: got %v want %v", eventKeys, wantKeys)
	}

	for stream, wantLastID := range map[string]int64{"members": 5002, "member_nicknames": 7003} {
		var cursor LegacyCustomerCursor
		if err := migrated.Where("stream = ?", stream).Take(&cursor).Error; err != nil {
			t.Fatalf("read %s cursor: %v", stream, err)
		}
		if cursor.Status != "complete" || cursor.LastLegacyID != wantLastID {
			t.Fatalf("unexpected %s cursor: %+v", stream, cursor)
		}
	}
	if !migrated.Migrator().HasTable("members") || !migrated.Migrator().HasTable("member_nicknames") {
		t.Fatal("legacy source tables were removed")
	}
	var schemaLedgerCount int64
	if err := migrated.Table("schema_migrations").Where("version IN ? AND status = ?", []uint{1, 2}, "applied").Count(&schemaLedgerCount).Error; err != nil {
		t.Fatalf("count applied schema migrations: %v", err)
	}
	if schemaLedgerCount != 2 {
		t.Fatalf("applied schema migration count = %d, want 2", schemaLedgerCount)
	}
	var dataLedger dataMigrationLedger
	if err := migrated.Where("version = ?", 1).Take(&dataLedger).Error; err != nil {
		t.Fatalf("read legacy data migration ledger: %v", err)
	}
	if dataLedger.Name != "legacy_customer_import_v1" || dataLedger.Status != "applied" ||
		dataLedger.Checkpoint != "complete" || dataLedger.RowsProcessed != 5 || len(dataLedger.Checksum) != 64 {
		t.Fatalf("unexpected legacy data migration ledger: %+v", dataLedger)
	}
	closeGormDB(t, migrated)

	reopened, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("reopen migrated database: %v", err)
	}
	defer closeGormDB(t, reopened)
	assertTableCount(t, reopened, "legacy_customer_maps", 1)
	assertTableCount(t, reopened, "customer_profiles", 1)
	assertTableCount(t, reopened, "customer_identities", 1)
	assertTableCount(t, reopened, "customer_name_observations", 2)
	assertTableCount(t, reopened, "customer_name_events", 2)
	assertTableCount(t, reopened, "legacy_customer_migration_quarantines", 2)
}

func TestLegacyCustomerMigrationRejectsBadSchema(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "bad-legacy-schema.db")
	database := openSilentSQLite(t, dbPath)
	if err := database.Exec(`CREATE TABLE members (id INTEGER PRIMARY KEY, platform TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("create bad legacy schema: %v", err)
	}
	closeGormDB(t, database)

	_, err := InitDB(dbPath)
	if err == nil || !strings.Contains(err.Error(), "members.platform_uid is required") {
		t.Fatalf("expected hard schema failure, got %v", err)
	}
}

func TestLegacyCustomerBatchFailureRollsBackAndRetries(t *testing.T) {
	t.Parallel()
	database := openSilentSQLite(t, filepath.Join(t.TempDir(), "legacy-rollback.db"))
	defer closeGormDB(t, database)
	if err := runSchemaMigrations(database, nil, customerResolutionMigrations()); err != nil {
		t.Fatalf("prepare customer resolution schema: %v", err)
	}
	createLegacyCustomerTables(t, database)
	if err := database.Exec(`INSERT INTO members (id, platform, platform_uid, nickname)
VALUES (9001, 'platform-a', 'legacy-user', 'Legacy User')`).Error; err != nil {
		t.Fatalf("seed legacy member: %v", err)
	}

	const callbackName = "test:fail_legacy_identity_create"
	err := database.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "customer_identities" {
			tx.AddError(errors.New("injected identity failure"))
		}
	})
	if err != nil {
		t.Fatalf("register failure callback: %v", err)
	}
	err = runBatchedDataMigrations(database, legacyCustomerDataMigrations())
	if err == nil || !strings.Contains(err.Error(), "injected identity failure") {
		t.Fatalf("expected injected batch failure, got %v", err)
	}
	assertTableCount(t, database, "customer_profiles", 0)
	assertTableCount(t, database, "customer_identities", 0)
	assertTableCount(t, database, "legacy_customer_maps", 0)
	assertTableCount(t, database, "legacy_customer_migration_cursors", 0)
	var failedLedger dataMigrationLedger
	if err := database.Where("version = ?", 1).Take(&failedLedger).Error; err != nil {
		t.Fatalf("read failed legacy data migration ledger: %v", err)
	}
	if failedLedger.Status != "failed" || !strings.Contains(failedLedger.ErrorMessage, "injected identity failure") {
		t.Fatalf("unexpected failed legacy data migration ledger: %+v", failedLedger)
	}
	if err := database.Callback().Create().Remove(callbackName); err != nil {
		t.Fatalf("remove failure callback: %v", err)
	}

	if err := runBatchedDataMigrations(database, legacyCustomerDataMigrations()); err != nil {
		t.Fatalf("retry legacy migration: %v", err)
	}
	assertTableCount(t, database, "customer_profiles", 1)
	assertTableCount(t, database, "customer_identities", 1)
	assertTableCount(t, database, "legacy_customer_maps", 1)
	var cursor LegacyCustomerCursor
	if err := database.Where("stream = ?", "members").Take(&cursor).Error; err != nil {
		t.Fatalf("read retried member cursor: %v", err)
	}
	if cursor.Status != "complete" || cursor.LastLegacyID != 9001 {
		t.Fatalf("unexpected retried member cursor: %+v", cursor)
	}
	var appliedLedger dataMigrationLedger
	if err := database.Where("version = ?", 1).Take(&appliedLedger).Error; err != nil {
		t.Fatalf("read recovered legacy data migration ledger: %v", err)
	}
	if appliedLedger.Status != "applied" || appliedLedger.Checkpoint != "complete" || appliedLedger.RowsProcessed != 1 {
		t.Fatalf("unexpected recovered legacy data migration ledger: %+v", appliedLedger)
	}
}

func TestLegacyAABHistoryAcceptsFirstProductionBOrC(t *testing.T) {
	t.Parallel()
	for _, nextName := range []string{"B", "C"} {
		nextName := nextName
		t.Run(nextName, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "legacy-aab.db")
			legacy := openSilentSQLite(t, dbPath)
			createLegacyCustomerTables(t, legacy)
			if err := legacy.Exec(`INSERT INTO members (id, platform, platform_uid, nickname)
VALUES (11001, 'legacy-platform', 'legacy-user', 'B')`).Error; err != nil {
				t.Fatalf("seed legacy member: %v", err)
			}
			for _, row := range []struct {
				id        int
				name      string
				createdAt string
			}{
				{id: 12001, name: "A", createdAt: "2024-01-01T00:00:00Z"},
				{id: 12002, name: "A", createdAt: "2024-01-02T00:00:00Z"},
				{id: 12003, name: "B", createdAt: "2024-01-03T00:00:00Z"},
			} {
				if err := legacy.Exec(`INSERT INTO member_nicknames (id, member_id, nickname, created_at)
VALUES (?, 11001, ?, ?)`, row.id, row.name, row.createdAt).Error; err != nil {
					t.Fatalf("seed legacy nickname: %v", err)
				}
			}
			closeGormDB(t, legacy)

			database, err := InitDB(dbPath)
			if err != nil {
				t.Fatalf("migrate legacy AAB history: %v", err)
			}
			defer closeGormDB(t, database)
			var mapping LegacyCustomerMap
			if err := database.Where("legacy_member_id = ?", 11001).Take(&mapping).Error; err != nil {
				t.Fatalf("read legacy map: %v", err)
			}

			var legacyObservations []struct {
				SourceEventKey string
				EpisodeKey     string
			}
			if err := database.Table("customer_name_observations").
				Select("source_event_key, episode_key").Order("id").Scan(&legacyObservations).Error; err != nil {
				t.Fatalf("read migrated observations: %v", err)
			}
			for _, observation := range legacyObservations {
				if observation.SourceEventKey == "" || observation.EpisodeKey != observation.SourceEventKey {
					t.Fatalf("legacy episode is incompatible: %+v", observation)
				}
			}
			var payloads []string
			if err := database.Table("customer_name_events").Where("event_key LIKE ?", "legacy:member_nicknames:%").
				Order("id").Pluck("payload", &payloads).Error; err != nil {
				t.Fatalf("read migrated event payloads: %v", err)
			}
			for _, rawPayload := range payloads {
				var payload domain.CustomerNameEventPayload
				if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
					t.Fatalf("decode shared name-event payload: %v", err)
				}
				if payload.NameKind != domain.CustomerNameKindTrustedNickname ||
					payload.Authority != "legacy_member_nicknames" || payload.TrustScore != 0.8 || payload.ExtraData == "" {
					t.Fatalf("legacy event did not use shared payload contract: %+v", payload)
				}
			}

			err = database.Transaction(func(tx *gorm.DB) error {
				service := app.NewCustomerNameObservationService(
					infra.NewProfileRepository(tx),
					infra.NewCustomerNameObservationRepository(tx),
					infra.NewCustomerNameEventRepository(tx),
				)
				_, err := service.Observe(context.Background(), app.ObserveCustomerNameInput{
					CustomerProfileID: mapping.CustomerProfileID,
					Name:              nextName,
					NameKind:          domain.CustomerNameKindStableIdentityNickname,
					Authority:         "production-import",
					TrustScore:        1,
					SourceEventKey:    "production:first:" + nextName,
					ObservedAt:        time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
				})
				return err
			})
			if err != nil {
				t.Fatalf("first production import %s after legacy AAB: %v", nextName, err)
			}

			observations, err := infra.NewCustomerNameObservationRepository(database).
				ListByProfile(context.Background(), mapping.CustomerProfileID)
			if err != nil {
				t.Fatalf("list rebuilt episodes: %v", err)
			}
			active := make([]domain.CustomerNameObservation, 0, len(observations))
			for _, observation := range observations {
				if observation.IsActive {
					active = append(active, observation)
				}
			}
			wantEpisodes := 2
			if nextName == "C" {
				wantEpisodes = 3
			}
			if len(active) != wantEpisodes || active[0].Name != "A" || active[0].ObservationCount != 2 {
				t.Fatalf("rebuilt %s episodes = %+v", nextName, active)
			}
			last := active[len(active)-1]
			if last.Name != nextName || (nextName == "B" && last.ObservationCount != 2) {
				t.Fatalf("last %s episode = %+v", nextName, last)
			}
			profile, err := infra.NewProfileRepository(database).FindByID(context.Background(), mapping.CustomerProfileID)
			if err != nil {
				t.Fatalf("read projected profile: %v", err)
			}
			if profile.DisplayName != nextName || profile.DisplayNameMode != domain.DisplayNameModeAuto {
				t.Fatalf("DisplayName after first %s import = %+v", nextName, profile)
			}
		})
	}
}

func createLegacyCustomerTables(t *testing.T, database *gorm.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE members (id INTEGER PRIMARY KEY, platform TEXT NOT NULL, platform_uid TEXT NOT NULL, nickname TEXT)`,
		`CREATE TABLE member_nicknames (id INTEGER PRIMARY KEY, member_id INTEGER NOT NULL, nickname TEXT, created_at DATETIME)`,
	} {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatalf("create legacy customer table: %v", err)
		}
	}
}

func openSilentSQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite %q: %v", path, err)
	}
	return database
}

func closeGormDB(t *testing.T, database *gorm.DB) {
	t.Helper()
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("get SQL DB for close: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close SQL DB: %v", err)
	}
}

func assertTableCount(t *testing.T, database *gorm.DB, table string, want int64) {
	t.Helper()
	var count int64
	if err := database.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}
