package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ domain.MergeGovernanceRepository = (*mergeGovernanceRepository)(nil)

type mergeGovernanceRepository struct{ db *gorm.DB }

func NewMergeGovernanceRepository(db *gorm.DB) domain.MergeGovernanceRepository {
	return &mergeGovernanceRepository{db: db}
}

func (r *mergeGovernanceRepository) EnsurePolicy(ctx context.Context, policy *domain.MergePolicy, revision *domain.MergePolicyRevision) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing persistence.MergePolicy
		err := tx.Where("policy_key = ?", policy.PolicyKey).First(&existing).Error
		if err == nil {
			*policy = *persistence.MergePolicyToDomain(&existing)
			if existing.CurrentRevisionID != nil {
				var current persistence.MergePolicyRevision
				if err := tx.First(&current, *existing.CurrentRevisionID).Error; err == nil &&
					isSafeSuggestRevision(current) && existing.DefaultAction == domain.MergePolicyActionSuggestOnly {
					*revision = *persistence.MergePolicyRevisionToDomain(&current)
					return nil
				} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}

			var maxRevision uint
			if err := tx.Model(&persistence.MergePolicyRevision{}).Where("merge_policy_id = ?", existing.ID).
				Select("COALESCE(MAX(revision), 0)").Scan(&maxRevision).Error; err != nil {
				return err
			}
			revision.MergePolicyID = existing.ID
			revision.Revision = maxRevision + 1
			revisionRow := persistence.MergePolicyRevisionFromDomain(revision)
			if err := tx.Create(revisionRow).Error; err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"current_revision_id": revisionRow.ID,
				"default_action":      domain.MergePolicyActionSuggestOnly,
				"row_version":         gorm.Expr("row_version + 1"),
				"needs_scan":          true,
			}).Error; err != nil {
				return err
			}
			existing.CurrentRevisionID = &revisionRow.ID
			*policy = *persistence.MergePolicyToDomain(&existing)
			*revision = *persistence.MergePolicyRevisionToDomain(revisionRow)
			created = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		row := persistence.MergePolicyFromDomain(policy)
		row.RowVersion = 1
		row.NeedsScan = true
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		revision.MergePolicyID = row.ID
		revisionRow := persistence.MergePolicyRevisionFromDomain(revision)
		if err := tx.Create(revisionRow).Error; err != nil {
			return err
		}
		if err := tx.Model(row).Updates(map[string]any{
			"current_revision_id": revisionRow.ID,
			"row_version":         1,
			"needs_scan":          true,
		}).Error; err != nil {
			return err
		}
		row.CurrentRevisionID = &revisionRow.ID
		*policy = *persistence.MergePolicyToDomain(row)
		*revision = *persistence.MergePolicyRevisionToDomain(revisionRow)
		created = true
		return nil
	})
	if err != nil && isUniqueConstraint(err) {
		_, _, findErr := r.FindPolicyByKey(ctx, policy.PolicyKey)
		if findErr == nil {
			return false, nil
		}
	}
	return created, err
}

func isSafeSuggestRevision(revision persistence.MergePolicyRevision) bool {
	if revision.Action != domain.MergePolicyActionSuggestOnly {
		return false
	}
	var rules domain.MergePolicyRules
	return json.Unmarshal([]byte(revision.Rules), &rules) == nil && rules.ExecutionMode == domain.MergePolicyActionSuggestOnly
}

func (r *mergeGovernanceRepository) FindPolicyByKey(ctx context.Context, policyKey string) (*domain.MergePolicy, *domain.MergePolicyRevision, error) {
	var policy persistence.MergePolicy
	if err := r.db.WithContext(ctx).Where("policy_key = ?", policyKey).First(&policy).Error; err != nil {
		return nil, nil, err
	}
	if policy.CurrentRevisionID == nil {
		return nil, nil, fmt.Errorf("merge policy %q has no current revision", policyKey)
	}
	var revision persistence.MergePolicyRevision
	if err := r.db.WithContext(ctx).First(&revision, *policy.CurrentRevisionID).Error; err != nil {
		return nil, nil, err
	}
	return persistence.MergePolicyToDomain(&policy), persistence.MergePolicyRevisionToDomain(&revision), nil
}

var errPolicyRevisionConflict = errors.New("merge policy revision conflict")

func (r *mergeGovernanceRepository) UpdatePolicyCAS(ctx context.Context, policyKey string, expectedRevision uint, revision *domain.MergePolicyRevision) (*domain.MergePolicy, bool, error) {
	var updated persistence.MergePolicy
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var policy persistence.MergePolicy
		if err := tx.Where("policy_key = ?", policyKey).First(&policy).Error; err != nil {
			return err
		}
		if policy.CurrentRevisionID == nil {
			return fmt.Errorf("merge policy %q has no current revision", policyKey)
		}
		var current persistence.MergePolicyRevision
		if err := tx.First(&current, *policy.CurrentRevisionID).Error; err != nil {
			return err
		}
		if current.Revision != expectedRevision {
			return errPolicyRevisionConflict
		}

		revision.MergePolicyID = policy.ID
		revision.Revision = expectedRevision + 1
		revisionRow := persistence.MergePolicyRevisionFromDomain(revision)
		if err := tx.Create(revisionRow).Error; err != nil {
			return err
		}
		result := tx.Model(&persistence.MergePolicy{}).
			Where("id = ? AND row_version = ?", policy.ID, policy.RowVersion).
			Updates(map[string]any{
				"current_revision_id": revisionRow.ID,
				"row_version":         gorm.Expr("row_version + 1"),
				"needs_scan":          true,
				"default_action":      domain.MergePolicyActionSuggestOnly,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errPolicyRevisionConflict
		}
		if err := tx.First(&updated, policy.ID).Error; err != nil {
			return err
		}
		*revision = *persistence.MergePolicyRevisionToDomain(revisionRow)
		return nil
	})
	if errors.Is(err, errPolicyRevisionConflict) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return persistence.MergePolicyToDomain(&updated), true, nil
}

func (r *mergeGovernanceRepository) CompletePolicyScan(ctx context.Context, policyID, policyRevisionID uint, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&persistence.MergePolicy{}).
		Where("id = ? AND current_revision_id = ?", policyID, policyRevisionID).
		Updates(map[string]any{"needs_scan": false, "last_scan_at": completedAt}).Error
}

func (r *mergeGovernanceRepository) CreateScanRun(ctx context.Context, run *domain.MergeScanRun) error {
	row := persistence.MergeScanRunFromDomain(run)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	*run = *persistence.MergeScanRunToDomain(row)
	return nil
}

func (r *mergeGovernanceRepository) UpdateScanRun(ctx context.Context, run *domain.MergeScanRun) error {
	if err := r.db.WithContext(ctx).Model(&persistence.MergeScanRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status": run.Status, "completed_at": run.CompletedAt,
		"profiles_scanned": run.ProfilesScanned, "pairs_evaluated": run.PairsEvaluated,
		"candidates_created": run.CandidatesCreated, "candidates_updated": run.CandidatesUpdated,
		"candidates_blocked": run.CandidatesBlocked, "error_message": run.ErrorMessage,
		"updated_at": run.UpdatedAt,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (r *mergeGovernanceRepository) FindScanRun(ctx context.Context, id uint) (*domain.MergeScanRun, error) {
	var row persistence.MergeScanRun
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.MergeScanRunToDomain(&row), nil
}

func (r *mergeGovernanceRepository) UpsertCandidateEvaluation(ctx context.Context, candidate *domain.MergeCandidate, evidence []domain.MergeEvidence) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row persistence.MergeCandidate
		err := tx.Where("canonical_pair_key = ? AND evidence_hash = ? AND policy_version = ?",
			candidate.CanonicalPairKey, candidate.EvidenceHash, candidate.PolicyVersion).First(&row).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			row = *persistence.MergeCandidateFromDomain(candidate)
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			created = true
		case err != nil:
			return err
		default:
			status := candidate.Status
			if row.Status == domain.MergeCandidateStatusDismissed || row.Status == domain.MergeCandidateStatusExecuted {
				status = row.Status
			}
			if err := tx.Model(&row).Updates(map[string]any{
				"source_profile_id": candidate.SourceProfileID, "target_profile_id": candidate.TargetProfileID,
				"status": status, "score": candidate.Score, "merge_policy_revision_id": candidate.MergePolicyRevisionID,
				"reason": candidate.Reason, "explanation_code": candidate.ExplanationCode,
				"confidence": candidate.Confidence, "blockers": candidate.Blockers,
				"last_evaluated_at": candidate.LastEvaluatedAt, "expires_at": candidate.ExpiresAt,
				"scan_run_id": candidate.ScanRunID, "row_version": gorm.Expr("row_version + 1"),
			}).Error; err != nil {
				return err
			}
			if err := tx.First(&row, row.ID).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&persistence.MergeCandidate{}).
			Where("canonical_pair_key = ? AND id <> ? AND status IN ?", candidate.CanonicalPairKey, row.ID,
				[]string{domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked}).
			Updates(map[string]any{"status": domain.MergeCandidateStatusSuperseded, "row_version": gorm.Expr("row_version + 1")}).Error; err != nil {
			return err
		}

		for i := range evidence {
			evidence[i].MergeCandidateID = row.ID
			evidenceRow := persistence.MergeEvidenceFromDomain(&evidence[i])
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(evidenceRow).Error; err != nil {
				return err
			}
		}
		*candidate = *persistence.MergeCandidateToDomain(&row)
		return nil
	})
	return created, err
}

func (r *mergeGovernanceRepository) MarkUnseenCandidatesStale(ctx context.Context, policyVersion, scanRunID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("(policy_version <> ? OR scan_run_id IS NULL OR scan_run_id <> ?) AND status IN ?", policyVersion, scanRunID,
			[]string{domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked}).
		Updates(map[string]any{"status": domain.MergeCandidateStatusStale, "row_version": gorm.Expr("row_version + 1")}).Error
}

func (r *mergeGovernanceRepository) FindCandidateWithEvidence(ctx context.Context, id uint) (*domain.MergeCandidate, []domain.MergeEvidence, error) {
	var candidate persistence.MergeCandidate
	if err := r.db.WithContext(ctx).First(&candidate, id).Error; err != nil {
		return nil, nil, err
	}
	var rows []persistence.MergeEvidence
	if err := r.db.WithContext(ctx).Where("merge_candidate_id = ?", id).Order("id").Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	evidence := make([]domain.MergeEvidence, len(rows))
	for i := range rows {
		evidence[i] = *persistence.MergeEvidenceToDomain(&rows[i])
	}
	return persistence.MergeCandidateToDomain(&candidate), evidence, nil
}

func (r *mergeGovernanceRepository) ListCandidates(ctx context.Context, status string) ([]domain.MergeCandidate, error) {
	query := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var rows []persistence.MergeCandidate
	if err := query.Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.MergeCandidate, len(rows))
	for i := range rows {
		result[i] = *persistence.MergeCandidateToDomain(&rows[i])
	}
	return result, nil
}

func (r *mergeGovernanceRepository) DismissCandidate(ctx context.Context, id uint, evidenceHash string, policyVersion uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("id = ? AND evidence_hash = ? AND policy_version = ? AND status IN ?", id, evidenceHash, policyVersion,
			[]string{domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked}).
		Updates(map[string]any{"status": domain.MergeCandidateStatusDismissed, "row_version": gorm.Expr("row_version + 1")})
	return result.RowsAffected == 1, result.Error
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return containsFold(message, "unique constraint") || containsFold(message, "duplicate key")
}

func containsFold(value, fragment string) bool {
	if len(fragment) > len(value) {
		return false
	}
	for i := 0; i+len(fragment) <= len(value); i++ {
		match := true
		for j := range fragment {
			a, b := value[i+j], fragment[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
