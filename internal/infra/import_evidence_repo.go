package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type importEvidenceRepository struct{ db *gorm.DB }

func NewImportEvidenceRepository(db *gorm.DB) domain.ImportEvidenceRepository {
	return &importEvidenceRepository{db: db}
}

func (r *importEvidenceRepository) GetSetting(ctx context.Context) (*domain.ImportEvidenceSetting, error) {
	var row persistence.ImportEvidenceSetting
	if err := r.db.WithContext(ctx).First(&row, 1).Error; err != nil {
		return nil, err
	}
	return &domain.ImportEvidenceSetting{ID: row.ID, RetentionDays: row.RetentionDays, Revision: row.Revision, UpdatedAt: row.UpdatedAt}, nil
}

func (r *importEvidenceRepository) SaveSetting(ctx context.Context, setting *domain.ImportEvidenceSetting) error {
	row := persistence.ImportEvidenceSetting{ID: 1, RetentionDays: setting.RetentionDays, Revision: setting.Revision, UpdatedAt: setting.UpdatedAt}
	if err := r.db.WithContext(ctx).Save(&row).Error; err != nil {
		return err
	}
	setting.ID = row.ID
	return nil
}

func (r *importEvidenceRepository) CreateRun(ctx context.Context, run *domain.ImportRun) error {
	row := importRunPersistence(run)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	run.ID = row.ID
	return nil
}

func (r *importEvidenceRepository) UpdateRun(ctx context.Context, run *domain.ImportRun) error {
	return r.db.WithContext(ctx).Save(importRunPersistence(run)).Error
}

func (r *importEvidenceRepository) CreateRecord(ctx context.Context, record *domain.ImportRawRecord) error {
	row := importRawRecordPersistence(record)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	record.ID = row.ID
	return nil
}

func (r *importEvidenceRepository) UpdateRecord(ctx context.Context, record *domain.ImportRawRecord) error {
	return r.db.WithContext(ctx).Save(importRawRecordPersistence(record)).Error
}

func (r *importEvidenceRepository) CreateRunWithRecords(ctx context.Context, run *domain.ImportRun, records []domain.ImportRawRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := importRunPersistence(run)
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		run.ID = row.ID
		for i := range records {
			records[i].ImportRunID = run.ID
			record := importRawRecordPersistence(&records[i])
			if err := tx.Create(record).Error; err != nil {
				return err
			}
			records[i].ID = record.ID
		}
		run.RecordCount = len(records)
		return tx.Save(importRunPersistence(run)).Error
	})
}

func (r *importEvidenceRepository) FinalizeRunWithRecords(ctx context.Context, run *domain.ImportRun, records []domain.ImportRawRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range records {
			if records[i].ID == 0 || records[i].ImportRunID != run.ID {
				return fmt.Errorf("finalize import evidence: invalid raw record identity at index %d", i)
			}
			result := tx.Model(&persistence.ImportRawRecord{}).
				Where("id = ? AND import_run_id = ?", records[i].ID, run.ID).
				Select("*").Updates(importRawRecordPersistence(&records[i]))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("finalize import evidence: raw record %d is missing", records[i].ID)
			}
		}
		result := tx.Model(&persistence.ImportRun{}).Where("id = ?", run.ID).Select("*").Updates(importRunPersistence(run))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("finalize import evidence: run %d is missing", run.ID)
		}
		return nil
	})
}

func (r *importEvidenceRepository) ListRuns(ctx context.Context, limit int) ([]domain.ImportRun, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return r.ListRunsPage(ctx, domain.ImportRunListQuery{Limit: limit})
}

func (r *importEvidenceRepository) ListRunsPage(ctx context.Context, query domain.ImportRunListQuery) ([]domain.ImportRun, error) {
	db := r.db.WithContext(ctx)
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.ProfileID != nil {
		db = db.Where("integration_profile_id = ?", *query.ProfileID)
	}
	if query.DocumentType != "" {
		db = db.Where("import_kind = ?", query.DocumentType)
	}
	if query.BeforeCreatedAt != nil {
		db = db.Where("created_at < ? OR (created_at = ? AND id < ?)", *query.BeforeCreatedAt, *query.BeforeCreatedAt, query.BeforeID)
	}
	var rows []persistence.ImportRun
	if err := db.Order("created_at DESC, id DESC").Limit(query.Limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ImportRun, len(rows))
	for i := range rows {
		out[i] = *importRunDomain(&rows[i])
	}
	return out, nil
}

func (r *importEvidenceRepository) FindRunByID(ctx context.Context, id uint) (*domain.ImportRun, error) {
	var row persistence.ImportRun
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return importRunDomain(&row), nil
}

func (r *importEvidenceRepository) ListRecordsByRun(ctx context.Context, runID uint) ([]domain.ImportRawRecord, error) {
	var rows []persistence.ImportRawRecord
	if err := r.db.WithContext(ctx).Where("import_run_id = ?", runID).Order("row_index, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ImportRawRecord, len(rows))
	for i := range rows {
		out[i] = *importRawRecordDomain(&rows[i])
	}
	return out, nil
}

func (r *importEvidenceRepository) PruneExpired(ctx context.Context, now time.Time) (int64, int64, error) {
	var runsDeleted, recordsDeleted int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		records := tx.Where("expires_at IS NOT NULL AND expires_at <= ?", now).Delete(&persistence.ImportRawRecord{})
		if records.Error != nil {
			return records.Error
		}
		recordsDeleted = records.RowsAffected
		runs := tx.Where("expires_at IS NOT NULL AND expires_at <= ?", now).Delete(&persistence.ImportRun{})
		if runs.Error != nil {
			return runs.Error
		}
		runsDeleted = runs.RowsAffected
		return nil
	})
	return runsDeleted, recordsDeleted, err
}

func importRunPersistence(d *domain.ImportRun) *persistence.ImportRun {
	return &persistence.ImportRun{ID: d.ID, RunKey: d.RunKey, ImportKind: d.ImportKind, IntegrationProfileID: d.IntegrationProfileID, SourceFormat: d.SourceFormat, SourceFileName: d.SourceFileName, ImportMode: d.ImportMode, Status: d.Status, RetentionDays: d.RetentionDays, RetentionPolicyVersion: d.RetentionPolicyVersion, ExpiresAt: d.ExpiresAt, RecordCount: d.RecordCount, SuccessCount: d.SuccessCount, FailureCount: d.FailureCount, QuarantinedCount: d.QuarantinedCount, ParserMetadata: d.ParserMetadata, CreatedAt: d.CreatedAt, CompletedAt: d.CompletedAt}
}
func importRunDomain(p *persistence.ImportRun) *domain.ImportRun {
	return &domain.ImportRun{ID: p.ID, RunKey: p.RunKey, ImportKind: p.ImportKind, IntegrationProfileID: p.IntegrationProfileID, SourceFormat: p.SourceFormat, SourceFileName: p.SourceFileName, ImportMode: p.ImportMode, Status: p.Status, RetentionDays: p.RetentionDays, RetentionPolicyVersion: p.RetentionPolicyVersion, ExpiresAt: p.ExpiresAt, RecordCount: p.RecordCount, SuccessCount: p.SuccessCount, FailureCount: p.FailureCount, QuarantinedCount: p.QuarantinedCount, ParserMetadata: p.ParserMetadata, CreatedAt: p.CreatedAt, CompletedAt: p.CompletedAt}
}
func importRawRecordPersistence(d *domain.ImportRawRecord) *persistence.ImportRawRecord {
	return &persistence.ImportRawRecord{ID: d.ID, ImportRunID: d.ImportRunID, RowIndex: d.RowIndex, RawLogicalRow: d.RawLogicalRow, UnmappedSource: d.UnmappedSource, ParserMetadata: d.ParserMetadata, WarningCodes: d.WarningCodes, AssetMembers: d.AssetMembers, Outcome: d.Outcome, ErrorCode: d.ErrorCode, ErrorMessage: d.ErrorMessage, ResultType: d.ResultType, ResultID: d.ResultID, RetentionDays: d.RetentionDays, ExpiresAt: d.ExpiresAt, CreatedAt: d.CreatedAt}
}
func importRawRecordDomain(p *persistence.ImportRawRecord) *domain.ImportRawRecord {
	return &domain.ImportRawRecord{ID: p.ID, ImportRunID: p.ImportRunID, RowIndex: p.RowIndex, RawLogicalRow: p.RawLogicalRow, UnmappedSource: p.UnmappedSource, ParserMetadata: p.ParserMetadata, WarningCodes: p.WarningCodes, AssetMembers: p.AssetMembers, Outcome: p.Outcome, ErrorCode: p.ErrorCode, ErrorMessage: p.ErrorMessage, ResultType: p.ResultType, ResultID: p.ResultID, RetentionDays: p.RetentionDays, ExpiresAt: p.ExpiresAt, CreatedAt: p.CreatedAt}
}

type externalCarrierRepository struct{ db *gorm.DB }

func NewExternalCarrierRepository(db *gorm.DB) domain.ExternalCarrierRepository {
	return &externalCarrierRepository{db: db}
}
func (r *externalCarrierRepository) Create(ctx context.Context, d *domain.ExternalCarrier) error {
	p := externalCarrierPersistence(d)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	d.ID = p.ID
	return nil
}
func (r *externalCarrierRepository) Update(ctx context.Context, d *domain.ExternalCarrier) error {
	return r.db.WithContext(ctx).Save(externalCarrierPersistence(d)).Error
}
func (r *externalCarrierRepository) FindByID(ctx context.Context, id uint) (*domain.ExternalCarrier, error) {
	var p persistence.ExternalCarrier
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return externalCarrierDomain(&p), nil
}
func (r *externalCarrierRepository) FindByCanonicalKey(ctx context.Context, profileID uint, key string) (*domain.ExternalCarrier, error) {
	var p persistence.ExternalCarrier
	err := r.db.WithContext(ctx).Where("integration_profile_id = ? AND canonical_key = ?", profileID, key).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return externalCarrierDomain(&p), nil
}
func (r *externalCarrierRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.ExternalCarrier, error) {
	var rows []persistence.ExternalCarrier
	if err := r.db.WithContext(ctx).Where("integration_profile_id = ?", profileID).Order("external_carrier_name, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ExternalCarrier, len(rows))
	for i := range rows {
		out[i] = *externalCarrierDomain(&rows[i])
	}
	return out, nil
}
func (r *externalCarrierRepository) CreateConflict(ctx context.Context, d *domain.ExternalCarrierConflict) error {
	p := &persistence.ExternalCarrierConflict{ID: d.ID, IntegrationProfileID: d.IntegrationProfileID, CanonicalKey: d.CanonicalKey, ConflictKind: d.ConflictKind, ExternalCarrierCode: d.ExternalCarrierCode, ExternalCarrierName: d.ExternalCarrierName, InternalCarrierCode: d.InternalCarrierCode, SourceImportRunID: d.SourceImportRunID, SourceRawRecordID: d.SourceRawRecordID, LegacyCarrierMappingID: d.LegacyCarrierMappingID, Payload: d.Payload, CreatedAt: d.CreatedAt}
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	d.ID = p.ID
	return nil
}
func (r *externalCarrierRepository) CreateConflicts(ctx context.Context, conflicts []domain.ExternalCarrierConflict) error {
	if len(conflicts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range conflicts {
			p := &persistence.ExternalCarrierConflict{ID: conflicts[i].ID, IntegrationProfileID: conflicts[i].IntegrationProfileID, CanonicalKey: conflicts[i].CanonicalKey, ConflictKind: conflicts[i].ConflictKind, ExternalCarrierCode: conflicts[i].ExternalCarrierCode, ExternalCarrierName: conflicts[i].ExternalCarrierName, InternalCarrierCode: conflicts[i].InternalCarrierCode, SourceImportRunID: conflicts[i].SourceImportRunID, SourceRawRecordID: conflicts[i].SourceRawRecordID, LegacyCarrierMappingID: conflicts[i].LegacyCarrierMappingID, Payload: conflicts[i].Payload, CreatedAt: conflicts[i].CreatedAt}
			if err := tx.Create(p).Error; err != nil {
				return err
			}
			conflicts[i].ID = p.ID
		}
		return nil
	})
}
func externalCarrierPersistence(d *domain.ExternalCarrier) *persistence.ExternalCarrier {
	return &persistence.ExternalCarrier{ID: d.ID, IntegrationProfileID: d.IntegrationProfileID, CanonicalKey: d.CanonicalKey, ExternalCarrierCode: d.ExternalCarrierCode, ExternalCarrierName: d.ExternalCarrierName, NameKeyStrategy: d.NameKeyStrategy, InternalCarrierCode: d.InternalCarrierCode, Status: d.Status, ConflictReason: d.ConflictReason, SourceImportRunID: d.SourceImportRunID, SourceRawRecordID: d.SourceRawRecordID, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}
func externalCarrierDomain(p *persistence.ExternalCarrier) *domain.ExternalCarrier {
	return &domain.ExternalCarrier{ID: p.ID, IntegrationProfileID: p.IntegrationProfileID, CanonicalKey: p.CanonicalKey, ExternalCarrierCode: p.ExternalCarrierCode, ExternalCarrierName: p.ExternalCarrierName, NameKeyStrategy: p.NameKeyStrategy, InternalCarrierCode: p.InternalCarrierCode, Status: p.Status, ConflictReason: p.ConflictReason, SourceImportRunID: p.SourceImportRunID, SourceRawRecordID: p.SourceRawRecordID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}
