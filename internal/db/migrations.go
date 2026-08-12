package db

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const migrationLedgerDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    applied_at DATETIME,
    error_message TEXT NOT NULL DEFAULT '',
    backup_path TEXT NOT NULL DEFAULT '',
    backup_checksum TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS data_migrations (
    version INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    checkpoint TEXT NOT NULL DEFAULT '',
    rows_processed INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME,
    applied_at DATETIME,
    error_message TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL
);`

type schemaMigrationLedger struct {
	Version        uint   `gorm:"primaryKey"`
	Name           string `gorm:"not null"`
	Checksum       string `gorm:"not null"`
	Status         string `gorm:"not null"`
	Attempt        uint   `gorm:"not null"`
	StartedAt      *time.Time
	AppliedAt      *time.Time
	ErrorMessage   string `gorm:"not null"`
	BackupPath     string `gorm:"not null"`
	BackupChecksum string `gorm:"not null"`
	UpdatedAt      time.Time
}

func (schemaMigrationLedger) TableName() string { return "schema_migrations" }

type dataMigrationLedger struct {
	Version       uint   `gorm:"primaryKey"`
	Name          string `gorm:"not null"`
	Checksum      string `gorm:"not null"`
	Status        string `gorm:"not null"`
	Checkpoint    string `gorm:"not null"`
	RowsProcessed uint64 `gorm:"not null"`
	StartedAt     *time.Time
	AppliedAt     *time.Time
	ErrorMessage  string `gorm:"not null"`
	UpdatedAt     time.Time
}

func (dataMigrationLedger) TableName() string { return "data_migrations" }

type schemaMigration struct {
	Version   uint
	Name      string
	Signature string
	Up        func(*gorm.DB) error
}

type dataMigration struct {
	Version   uint
	Name      string
	Signature string
	Up        func(*gorm.DB) (checkpoint string, rowsProcessed uint64, err error)
}

// batchedDataMigration is ledger-compatible with dataMigration but deliberately
// does not wrap the whole operation in one transaction. Its Up function owns
// durable batch transactions and cursors so large imports can resume safely.
type batchedDataMigration struct {
	Version   uint
	Name      string
	Signature string
	Up        func(*gorm.DB) (checkpoint string, rowsProcessed uint64, err error)
}

func (m batchedDataMigration) checksum() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", m.Version, m.Name, m.Signature)))
	return hex.EncodeToString(sum[:])
}

func (m dataMigration) checksum() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", m.Version, m.Name, m.Signature)))
	return hex.EncodeToString(sum[:])
}

func (m schemaMigration) checksum() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", m.Version, m.Name, m.Signature)))
	return hex.EncodeToString(sum[:])
}

func customerResolutionMigrations() []schemaMigration {
	return []schemaMigration{
		{
			Version:   1,
			Name:      "customer_resolution_foundation_v1",
			Signature: customerResolutionV1Signature(),
			Up:        applyCustomerResolutionV1,
		},
		{
			Version:   2,
			Name:      "customer_resolution_legacy_import_v2",
			Signature: customerResolutionV2Signature(),
			Up:        applyCustomerResolutionV2,
		},
		{
			Version:   3,
			Name:      "merge_governance_v3",
			Signature: customerResolutionV3Signature(),
			Up:        applyCustomerResolutionV3,
		},
		{
			Version:   4,
			Name:      "merge_execution_audit_v4",
			Signature: customerResolutionV4Signature(),
			Up:        applyCustomerResolutionV4,
		},
		{
			Version:   5,
			Name:      "customer_split_execution_audit_v5",
			Signature: customerResolutionV5Signature(),
			Up:        applyCustomerResolutionV5,
		},
		{
			Version:   6,
			Name:      "import_evidence_external_carrier_v6",
			Signature: importEvidenceCarrierV6Signature(),
			Up:        applyImportEvidenceCarrierV6,
		},
		{
			Version:   7,
			Name:      "schema_compatibility_audit_v7",
			Signature: schemaCompatibilityV7Signature(),
			Up:        applySchemaCompatibilityV7,
		},
		{
			Version:   8,
			Name:      "customer_resolution_feature_policy_v8",
			Signature: customerResolutionFeaturePolicyV8Signature(),
			Up:        applyCustomerResolutionFeaturePolicyV8,
		},
	}
}

func ensureMigrationLedgers(db *gorm.DB) error {
	if err := db.Exec(migrationLedgerDDL).Error; err != nil {
		return fmt.Errorf("create migration ledgers: %w", err)
	}
	return nil
}

func runSchemaMigrations(db *gorm.DB, backup *databaseBackup, migrations []schemaMigration) error {
	if err := ensureMigrationLedgers(db); err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version == 0 || migration.Name == "" || migration.Signature == "" || migration.Up == nil {
			return fmt.Errorf("invalid schema migration declaration at version %d", migration.Version)
		}
		checksum := migration.checksum()
		var ledger schemaMigrationLedger
		err := db.First(&ledger, "version = ?", migration.Version).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			now := time.Now().UTC()
			ledger = schemaMigrationLedger{
				Version: migration.Version, Name: migration.Name, Checksum: checksum,
				Status: "pending", ErrorMessage: "", UpdatedAt: now,
			}
			if backup != nil {
				ledger.BackupPath = backup.Path
				ledger.BackupChecksum = backup.Checksum
			}
			if err := db.Create(&ledger).Error; err != nil {
				return fmt.Errorf("register schema migration %d: %w", migration.Version, err)
			}
		case err != nil:
			return fmt.Errorf("read schema migration %d: %w", migration.Version, err)
		case ledger.Name != migration.Name || ledger.Checksum != checksum:
			return fmt.Errorf("schema migration %d checksum mismatch: ledger=%s/%s code=%s/%s",
				migration.Version, ledger.Name, ledger.Checksum, migration.Name, checksum)
		case ledger.Status == "applied":
			continue
		}

		startedAt := time.Now().UTC()
		updates := map[string]any{
			"status": "applying", "started_at": startedAt, "applied_at": nil,
			"error_message": "", "attempt": gorm.Expr("attempt + 1"), "updated_at": startedAt,
		}
		if backup != nil {
			updates["backup_path"] = backup.Path
			updates["backup_checksum"] = backup.Checksum
		}
		if err := db.Model(&schemaMigrationLedger{}).Where("version = ?", migration.Version).Updates(updates).Error; err != nil {
			return fmt.Errorf("start schema migration %d: %w", migration.Version, err)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return err
			}
			appliedAt := time.Now().UTC()
			return tx.Model(&schemaMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
				"status": "applied", "applied_at": appliedAt, "error_message": "", "updated_at": appliedAt,
			}).Error
		})
		if err != nil {
			failedAt := time.Now().UTC()
			failure := db.Model(&schemaMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
				"status": "failed", "error_message": err.Error(), "updated_at": failedAt,
			}).Error
			if failure != nil {
				return fmt.Errorf("schema migration %d failed: %v; persist failure state: %w", migration.Version, err, failure)
			}
			return fmt.Errorf("schema migration %d (%s) failed: %w", migration.Version, migration.Name, err)
		}
	}
	return nil
}

func runDataMigrations(db *gorm.DB, migrations []dataMigration) error {
	if err := ensureMigrationLedgers(db); err != nil {
		return err
	}

	for _, migration := range migrations {
		if migration.Version == 0 || migration.Name == "" || migration.Signature == "" || migration.Up == nil {
			return fmt.Errorf("invalid data migration declaration at version %d", migration.Version)
		}
		checksum := migration.checksum()
		var ledger dataMigrationLedger
		err := db.First(&ledger, "version = ?", migration.Version).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			now := time.Now().UTC()
			ledger = dataMigrationLedger{
				Version: migration.Version, Name: migration.Name, Checksum: checksum,
				Status: "pending", Checkpoint: "", ErrorMessage: "", UpdatedAt: now,
			}
			if err := db.Create(&ledger).Error; err != nil {
				return fmt.Errorf("register data migration %d: %w", migration.Version, err)
			}
		case err != nil:
			return fmt.Errorf("read data migration %d: %w", migration.Version, err)
		case ledger.Name != migration.Name || ledger.Checksum != checksum:
			return fmt.Errorf("data migration %d checksum mismatch: ledger=%s/%s code=%s/%s",
				migration.Version, ledger.Name, ledger.Checksum, migration.Name, checksum)
		case ledger.Status == "applied":
			continue
		}

		startedAt := time.Now().UTC()
		if err := db.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
			"status": "applying", "started_at": startedAt, "applied_at": nil,
			"error_message": "", "updated_at": startedAt,
		}).Error; err != nil {
			return fmt.Errorf("start data migration %d: %w", migration.Version, err)
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			checkpoint, rowsProcessed, err := migration.Up(tx)
			if err != nil {
				return err
			}
			appliedAt := time.Now().UTC()
			return tx.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
				"status": "applied", "checkpoint": checkpoint, "rows_processed": rowsProcessed,
				"applied_at": appliedAt, "error_message": "", "updated_at": appliedAt,
			}).Error
		})
		if err != nil {
			failedAt := time.Now().UTC()
			failure := db.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
				"status": "failed", "error_message": err.Error(), "updated_at": failedAt,
			}).Error
			if failure != nil {
				return fmt.Errorf("data migration %d failed: %v; persist failure state: %w", migration.Version, err, failure)
			}
			return fmt.Errorf("data migration %d (%s) failed: %w", migration.Version, migration.Name, err)
		}
	}
	return nil
}

func runBatchedDataMigrations(db *gorm.DB, migrations []batchedDataMigration) error {
	if err := ensureMigrationLedgers(db); err != nil {
		return err
	}
	for _, migration := range migrations {
		if migration.Version == 0 || migration.Name == "" || migration.Signature == "" || migration.Up == nil {
			return fmt.Errorf("invalid batched data migration declaration at version %d", migration.Version)
		}
		checksum := migration.checksum()
		var ledger dataMigrationLedger
		err := db.First(&ledger, "version = ?", migration.Version).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			now := time.Now().UTC()
			ledger = dataMigrationLedger{
				Version: migration.Version, Name: migration.Name, Checksum: checksum,
				Status: "pending", Checkpoint: "", ErrorMessage: "", UpdatedAt: now,
			}
			if err := db.Create(&ledger).Error; err != nil {
				return fmt.Errorf("register batched data migration %d: %w", migration.Version, err)
			}
		case err != nil:
			return fmt.Errorf("read batched data migration %d: %w", migration.Version, err)
		case ledger.Name != migration.Name || ledger.Checksum != checksum:
			return fmt.Errorf("batched data migration %d checksum mismatch: ledger=%s/%s code=%s/%s",
				migration.Version, ledger.Name, ledger.Checksum, migration.Name, checksum)
		case ledger.Status == "applied":
			continue
		}

		startedAt := time.Now().UTC()
		if err := db.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
			"status": "applying", "started_at": startedAt, "applied_at": nil,
			"error_message": "", "updated_at": startedAt,
		}).Error; err != nil {
			return fmt.Errorf("start batched data migration %d: %w", migration.Version, err)
		}

		checkpoint, rowsProcessed, err := migration.Up(db)
		if err != nil {
			failedAt := time.Now().UTC()
			failure := db.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
				"status": "failed", "error_message": err.Error(), "updated_at": failedAt,
			}).Error
			if failure != nil {
				return fmt.Errorf("batched data migration %d failed: %v; persist failure state: %w", migration.Version, err, failure)
			}
			return fmt.Errorf("batched data migration %d (%s) failed: %w", migration.Version, migration.Name, err)
		}
		appliedAt := time.Now().UTC()
		if err := db.Model(&dataMigrationLedger{}).Where("version = ?", migration.Version).Updates(map[string]any{
			"status": "applied", "checkpoint": checkpoint, "rows_processed": rowsProcessed,
			"applied_at": appliedAt, "error_message": "", "updated_at": appliedAt,
		}).Error; err != nil {
			return fmt.Errorf("complete batched data migration %d: %w", migration.Version, err)
		}
	}
	return nil
}
