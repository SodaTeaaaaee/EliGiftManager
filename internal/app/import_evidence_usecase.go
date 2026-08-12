package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ImportEvidenceUseCase struct {
	repo            domain.ImportEvidenceRepository
	deferred        bool
	initialized     bool
	disabled        bool
	preparedRun     *domain.ImportRun
	preparedRecords []domain.ImportRawRecord
	pendingRun      *domain.ImportRun
	pendingRecords  []domain.ImportRawRecord
}

func NewImportEvidenceUseCase(repo domain.ImportEvidenceRepository) *ImportEvidenceUseCase {
	return &ImportEvidenceUseCase{repo: repo}
}

// NewDeferredImportEvidenceUseCase creates an evidence coordinator for imports
// whose business writes run in a separate transaction. StartImportEvidence must
// be called before the business transaction opens; CompleteImportEvidence then
// stages the final state in memory and FinalizePending persists it afterwards.
func NewDeferredImportEvidenceUseCase(repo domain.ImportEvidenceRepository) *ImportEvidenceUseCase {
	return &ImportEvidenceUseCase{repo: repo, deferred: true}
}

func (uc *ImportEvidenceUseCase) Enabled(ctx context.Context) (bool, error) {
	if uc.initialized {
		return !uc.disabled, nil
	}
	enabled, err := customerResolutionFeatureEnabled(ctx, uc.repo, domain.CustomerResolutionFeatureImportEvidence)
	if err != nil {
		return false, err
	}
	uc.initialized = true
	uc.disabled = !enabled
	return enabled, nil
}

// ImportEvidenceAuditIncompleteError means the business outcome is already
// decided but the independent audit finalization could not be persisted.
type ImportEvidenceAuditIncompleteError struct {
	RunID uint
	Err   error
}

func (e *ImportEvidenceAuditIncompleteError) Error() string {
	return fmt.Sprintf("import evidence audit incomplete for run %d: %v", e.RunID, e.Err)
}

func (e *ImportEvidenceAuditIncompleteError) Unwrap() error { return e.Err }

func (uc *ImportEvidenceUseCase) GetRetention(ctx context.Context) (dto.ImportEvidenceRetentionDTO, error) {
	s, err := uc.repo.GetSetting(ctx)
	if err != nil {
		return dto.ImportEvidenceRetentionDTO{}, err
	}
	return dto.ImportEvidenceRetentionDTO{RetentionDays: s.RetentionDays, Revision: s.Revision, UpdatedAt: s.UpdatedAt}, nil
}
func (uc *ImportEvidenceUseCase) SetRetention(ctx context.Context, input dto.SetImportEvidenceRetentionInput) (dto.ImportEvidenceRetentionDTO, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureImportEvidence); err != nil {
		return dto.ImportEvidenceRetentionDTO{}, err
	}
	if !validImportRetention(input.RetentionDays) {
		return dto.ImportEvidenceRetentionDTO{}, fmt.Errorf("retentionDays must be 0, 30, 90, or -1 (permanent)")
	}
	s, err := uc.repo.GetSetting(ctx)
	if err != nil {
		return dto.ImportEvidenceRetentionDTO{}, err
	}
	s.RetentionDays = input.RetentionDays
	s.Revision++
	s.UpdatedAt = time.Now().UTC()
	if err := uc.repo.SaveSetting(ctx, s); err != nil {
		return dto.ImportEvidenceRetentionDTO{}, err
	}
	return dto.ImportEvidenceRetentionDTO{RetentionDays: s.RetentionDays, Revision: s.Revision, UpdatedAt: s.UpdatedAt}, nil
}
func validImportRetention(days int) bool {
	return days == domain.ImportRetentionImmediate || days == domain.ImportRetention30Days || days == domain.ImportRetention90Days || days == domain.ImportRetentionPermanent
}
func (uc *ImportEvidenceUseCase) PruneExpired(ctx context.Context) (int64, int64, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureImportEvidence); err != nil {
		return 0, 0, err
	}
	return uc.repo.PruneExpired(ctx, time.Now().UTC())
}
func (uc *ImportEvidenceUseCase) ListRuns(ctx context.Context, limit int) ([]dto.ImportRunSummaryDTO, error) {
	runs, err := uc.repo.ListRuns(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ImportRunSummaryDTO, len(runs))
	for i := range runs {
		out[i] = importRunSummaryDTO(&runs[i])
	}
	return out, nil
}

const (
	defaultImportRunPageLimit = 100
	maxImportRunPageLimit     = 500
	maxImportRunCursorLength  = 1024
)

type importRunCursor struct {
	Version     int    `json:"v"`
	CreatedAt   string `json:"createdAt"`
	ID          uint   `json:"id"`
	FilterToken string `json:"filter"`
}

func (uc *ImportEvidenceUseCase) ListRunsPage(ctx context.Context, input dto.ListImportRunsPageInput) (dto.ImportRunPageDTO, error) {
	query, limit, filterToken, err := importRunPageQuery(input)
	if err != nil {
		return dto.ImportRunPageDTO{}, err
	}
	if input.Cursor != "" {
		createdAt, id, err := decodeImportRunCursor(input.Cursor, filterToken)
		if err != nil {
			return dto.ImportRunPageDTO{}, err
		}
		query.BeforeCreatedAt = &createdAt
		query.BeforeID = id
	}
	query.Limit = limit + 1
	runs, err := uc.repo.ListRunsPage(ctx, query)
	if err != nil {
		return dto.ImportRunPageDTO{}, err
	}
	hasMore := len(runs) > limit
	if hasMore {
		runs = runs[:limit]
	}
	items := make([]dto.ImportRunSummaryDTO, len(runs))
	for i := range runs {
		items[i] = importRunSummaryDTO(&runs[i])
	}
	page := dto.ImportRunPageDTO{Items: items, HasMore: hasMore}
	if hasMore && len(runs) != 0 {
		page.NextCursor, err = encodeImportRunCursor(runs[len(runs)-1], filterToken)
		if err != nil {
			return dto.ImportRunPageDTO{}, err
		}
	}
	return page, nil
}

func importRunPageQuery(input dto.ListImportRunsPageInput) (domain.ImportRunListQuery, int, string, error) {
	limit := input.Limit
	if limit == 0 {
		limit = defaultImportRunPageLimit
	}
	if limit < 1 || limit > maxImportRunPageLimit {
		return domain.ImportRunListQuery{}, 0, "", fmt.Errorf("limit must be between 1 and %d", maxImportRunPageLimit)
	}
	status := strings.TrimSpace(input.Status)
	documentType := strings.TrimSpace(input.DocumentType)
	if input.Status != status || input.DocumentType != documentType {
		return domain.ImportRunListQuery{}, 0, "", fmt.Errorf("status and documentType must not contain surrounding whitespace")
	}
	if input.ProfileID != nil && *input.ProfileID == 0 {
		return domain.ImportRunListQuery{}, 0, "", fmt.Errorf("profileId must be greater than zero")
	}
	query := domain.ImportRunListQuery{Status: status, ProfileID: input.ProfileID, DocumentType: documentType}
	return query, limit, importRunFilterToken(query), nil
}

func importRunFilterToken(query domain.ImportRunListQuery) string {
	profileID := ""
	if query.ProfileID != nil {
		profileID = strconv.FormatUint(uint64(*query.ProfileID), 10)
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{"v1", query.Status, profileID, query.DocumentType}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func encodeImportRunCursor(run domain.ImportRun, filterToken string) (string, error) {
	payload, err := json.Marshal(importRunCursor{
		Version:     1,
		CreatedAt:   run.CreatedAt.UTC().Format(time.RFC3339Nano),
		ID:          run.ID,
		FilterToken: filterToken,
	})
	if err != nil {
		return "", fmt.Errorf("encode import runs cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeImportRunCursor(raw, filterToken string) (time.Time, uint, error) {
	invalid := func(reason string) (time.Time, uint, error) {
		return time.Time{}, 0, fmt.Errorf("invalid import runs cursor: %s", reason)
	}
	if raw == "" || strings.TrimSpace(raw) != raw || len(raw) > maxImportRunCursorLength {
		return invalid("malformed value")
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil {
		return invalid("malformed base64url")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var cursor importRunCursor
	if err := decoder.Decode(&cursor); err != nil {
		return invalid("malformed payload")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return invalid("malformed payload")
	}
	if cursor.Version != 1 || cursor.ID == 0 || cursor.FilterToken != filterToken {
		return invalid("version, position, or filters do not match")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
	if err != nil || createdAt.IsZero() || cursor.CreatedAt != createdAt.UTC().Format(time.RFC3339Nano) {
		return invalid("createdAt is not canonical UTC RFC3339")
	}
	return createdAt, cursor.ID, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("unexpected trailing JSON value")
	}
	return err
}

func (uc *ImportEvidenceUseCase) GetRunDetail(ctx context.Context, id uint) (dto.ImportRunDetailDTO, error) {
	run, err := uc.repo.FindRunByID(ctx, id)
	if err != nil {
		return dto.ImportRunDetailDTO{}, err
	}
	records, err := uc.repo.ListRecordsByRun(ctx, id)
	if err != nil {
		return dto.ImportRunDetailDTO{}, err
	}
	out := dto.ImportRunDetailDTO{Run: importRunSummaryDTO(run), Records: make([]dto.ImportRawRecordDetailDTO, len(records))}
	for i, r := range records {
		out.Records[i] = dto.ImportRawRecordDetailDTO{ID: r.ID, RowIndex: r.RowIndex, RawLogicalRow: r.RawLogicalRow, UnmappedSource: r.UnmappedSource, ParserMetadata: r.ParserMetadata, WarningCodes: r.WarningCodes, AssetMembers: r.AssetMembers, Outcome: r.Outcome, ErrorCode: r.ErrorCode, ErrorMessage: r.ErrorMessage, ResultType: r.ResultType, ResultID: r.ResultID, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// StartImportEvidence captures logical rows. Callers may pass ordered JSON-safe
// row values; ZIP callers pass only the tabular rows and asset member metadata,
// never binary content.
func (uc *ImportEvidenceUseCase) StartImportEvidence(ctx context.Context, kind string, profileID uint, mode, sourcePath, parserMetadata string, rows []any, unmapped []map[string]string, assets [][]map[string]string) (*domain.ImportRun, []domain.ImportRawRecord, error) {
	if uc.deferred && uc.initialized {
		if uc.disabled {
			return nil, nil, nil
		}
		if uc.preparedRun != nil {
			if err := validatePreparedEvidence(uc.preparedRun, kind, profileID, mode, sourcePath); err != nil {
				return nil, nil, err
			}
			return uc.preparedRun, uc.preparedRecords, nil
		}
	}
	enabled, err := uc.Enabled(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		return nil, nil, nil
	}
	setting, err := uc.repo.GetSetting(ctx)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	expires := importExpiry(now, setting.RetentionDays)
	run := &domain.ImportRun{RunKey: newImportRunKey(), ImportKind: kind, ImportMode: mode, SourceFormat: strings.TrimPrefix(strings.ToLower(filepath.Ext(sourcePath)), "."), SourceFileName: filepath.Base(sourcePath), Status: "running", RetentionDays: setting.RetentionDays, RetentionPolicyVersion: setting.Revision, ExpiresAt: expires, ParserMetadata: parserMetadata, CreatedAt: now}
	if profileID != 0 {
		run.IntegrationProfileID = &profileID
	}
	records := make([]domain.ImportRawRecord, len(rows))
	for i, row := range rows {
		raw, _ := json.Marshal(row)
		unmappedRaw := []byte("{}")
		if i < len(unmapped) {
			unmappedRaw, _ = json.Marshal(unmapped[i])
		}
		assetRaw := []byte("[]")
		if i < len(assets) {
			assetRaw, _ = json.Marshal(assets[i])
		}
		records[i] = domain.ImportRawRecord{RowIndex: i, RawLogicalRow: string(raw), UnmappedSource: string(unmappedRaw), ParserMetadata: parserMetadata, WarningCodes: "[]", AssetMembers: string(assetRaw), Outcome: "pending", RetentionDays: setting.RetentionDays, ExpiresAt: expires, CreatedAt: now}
	}
	run.RecordCount = len(records)
	if err := uc.repo.CreateRunWithRecords(ctx, run, records); err != nil {
		return nil, nil, err
	}
	if uc.deferred {
		uc.preparedRun = run
		uc.preparedRecords = records
	}
	return run, records, nil
}
func (uc *ImportEvidenceUseCase) CompleteImportEvidence(ctx context.Context, run *domain.ImportRun, records []domain.ImportRawRecord, status string) error {
	if run == nil {
		return nil
	}
	if !uc.deferred {
		enabled, err := customerResolutionFeatureEnabled(ctx, uc.repo, domain.CustomerResolutionFeatureImportEvidence)
		if err != nil {
			return err
		}
		if !enabled {
			return nil
		}
	} else if !uc.initialized || uc.disabled {
		if uc.disabled {
			return nil
		}
		return fmt.Errorf("import evidence was not prepared before the business transaction")
	}
	completeImportEvidenceState(run, records, status, time.Now().UTC())
	if uc.deferred {
		uc.pendingRun = run
		uc.pendingRecords = records
		return nil
	}
	return uc.repo.FinalizeRunWithRecords(ctx, run, records)
}

// FinalizePending atomically persists the final evidence state after the
// business transaction has committed.
func (uc *ImportEvidenceUseCase) FinalizePending(ctx context.Context) error {
	if !uc.deferred || uc.disabled || uc.preparedRun == nil {
		return nil
	}
	if uc.pendingRun == nil {
		return &ImportEvidenceAuditIncompleteError{RunID: uc.preparedRun.ID, Err: fmt.Errorf("final outcome was not staged")}
	}
	if err := uc.repo.FinalizeRunWithRecords(ctx, uc.pendingRun, uc.pendingRecords); err != nil {
		return &ImportEvidenceAuditIncompleteError{RunID: uc.preparedRun.ID, Err: err}
	}
	return nil
}

// FinalizeFailure records an independently durable failed or rejected outcome
// after the business transaction has rolled back. Any staged successes are
// invalidated because their result rows did not commit.
func (uc *ImportEvidenceUseCase) FinalizeFailure(ctx context.Context, status string, cause error) error {
	if !uc.deferred || uc.disabled || uc.preparedRun == nil {
		return nil
	}
	records := uc.preparedRecords
	if uc.pendingRecords != nil {
		records = uc.pendingRecords
	}
	code := "business_rollback"
	message := "business transaction rolled back"
	if status == "rejected" {
		code = "import_rejected"
		message = "import rejected before business writes"
	}
	if cause != nil {
		message = cause.Error()
	}
	for i := range records {
		if records[i].Outcome == "pending" || records[i].Outcome == "success" {
			records[i].Outcome = "failed"
			records[i].ErrorCode = code
			records[i].ErrorMessage = message
			records[i].ResultType = ""
			records[i].ResultID = nil
		}
	}
	completeImportEvidenceState(uc.preparedRun, records, status, time.Now().UTC())
	if err := uc.repo.FinalizeRunWithRecords(ctx, uc.preparedRun, records); err != nil {
		return &ImportEvidenceAuditIncompleteError{RunID: uc.preparedRun.ID, Err: err}
	}
	return nil
}

func (uc *ImportEvidenceUseCase) PreparedRunID() uint {
	if uc.preparedRun == nil {
		return 0
	}
	return uc.preparedRun.ID
}

func completeImportEvidenceState(run *domain.ImportRun, records []domain.ImportRawRecord, status string, completedAt time.Time) {
	run.Status = status
	run.CompletedAt = &completedAt
	run.SuccessCount = 0
	run.FailureCount = 0
	run.QuarantinedCount = 0
	for i := range records {
		switch records[i].Outcome {
		case "success":
			run.SuccessCount++
		case "quarantined", "review":
			run.QuarantinedCount++
		default:
			run.FailureCount++
		}
	}
}

func validatePreparedEvidence(run *domain.ImportRun, kind string, profileID uint, mode, sourcePath string) error {
	if run.ImportKind != kind || run.ImportMode != mode || run.SourceFileName != filepath.Base(sourcePath) {
		return fmt.Errorf("prepared import evidence does not match business import")
	}
	if profileID == 0 {
		if run.IntegrationProfileID != nil {
			return fmt.Errorf("prepared import evidence profile does not match business import")
		}
		return nil
	}
	if run.IntegrationProfileID == nil || *run.IntegrationProfileID != profileID {
		return fmt.Errorf("prepared import evidence profile does not match business import")
	}
	return nil
}
func importExpiry(now time.Time, days int) *time.Time {
	if days < 0 {
		return nil
	}
	v := now.Add(time.Duration(days) * 24 * time.Hour)
	return &v
}
func newImportRunKey() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}
func importRunSummaryDTO(r *domain.ImportRun) dto.ImportRunSummaryDTO {
	return dto.ImportRunSummaryDTO{ID: r.ID, ImportKind: r.ImportKind, IntegrationProfileID: r.IntegrationProfileID, SourceFormat: r.SourceFormat, SourceFileName: r.SourceFileName, ImportMode: r.ImportMode, Status: r.Status, RetentionDays: r.RetentionDays, RetentionPolicyVersion: r.RetentionPolicyVersion, ExpiresAt: r.ExpiresAt, RecordCount: r.RecordCount, SuccessCount: r.SuccessCount, FailureCount: r.FailureCount, QuarantinedCount: r.QuarantinedCount, CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt}
}
