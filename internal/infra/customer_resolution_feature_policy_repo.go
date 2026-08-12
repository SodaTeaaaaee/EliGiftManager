package infra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type customerResolutionFeaturePolicyRepository struct{ db *gorm.DB }

var errFeaturePolicyCASConflict = errors.New("customer resolution feature policy CAS conflict")

var _ domain.CustomerResolutionFeaturePolicyRepository = (*customerResolutionFeaturePolicyRepository)(nil)

func NewCustomerResolutionFeaturePolicyRepository(db *gorm.DB) domain.CustomerResolutionFeaturePolicyRepository {
	return &customerResolutionFeaturePolicyRepository{db: db}
}

func (r *customerResolutionFeaturePolicyRepository) GetFeaturePolicy(ctx context.Context) (*domain.CustomerResolutionFeaturePolicy, error) {
	var row persistence.CustomerResolutionFeaturePolicy
	if err := r.db.WithContext(ctx).First(&row, 1).Error; err != nil {
		return nil, featurePolicyUnavailable(err)
	}
	return featurePolicyToDomain(&row), nil
}

func (r *customerResolutionFeaturePolicyRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	policy, err := r.GetFeaturePolicy(ctx)
	if err != nil {
		return false, err
	}
	switch feature {
	case domain.CustomerResolutionFeatureWrites:
		return policy.CustomerResolutionWritesEnabled, nil
	case domain.CustomerResolutionFeatureCandidateScan:
		return policy.CustomerResolutionWritesEnabled && policy.CandidateScanEnabled, nil
	case domain.CustomerResolutionFeatureMergeExecution:
		return policy.CustomerResolutionWritesEnabled && policy.MergeExecutionEnabled, nil
	case domain.CustomerResolutionFeatureSplitExecution:
		return policy.CustomerResolutionWritesEnabled && policy.SplitExecutionEnabled, nil
	case domain.CustomerResolutionFeatureImportEvidence:
		return policy.CustomerResolutionWritesEnabled && policy.ImportEvidenceEnabled, nil
	case domain.CustomerResolutionFeatureCarrierRegistry:
		return policy.CustomerResolutionWritesEnabled && policy.CarrierRegistryWritesEnabled, nil
	default:
		return false, &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeUnavailable, Feature: feature, Detail: "unknown customer resolution feature"}
	}
}

func (r *customerResolutionFeaturePolicyRepository) RequireFeature(ctx context.Context, feature string) error {
	enabled, err := r.FeatureEnabled(ctx, feature)
	if err != nil {
		return err
	}
	if !enabled {
		return domain.FeaturePolicyDisabledError(feature)
	}
	return nil
}

func (r *customerResolutionFeaturePolicyRepository) UpdateFeaturePolicyCAS(
	ctx context.Context,
	expectedRevision uint64,
	next *domain.CustomerResolutionFeaturePolicy,
) (*domain.CustomerResolutionFeaturePolicy, bool, error) {
	if next == nil || expectedRevision == 0 {
		return nil, false, fmt.Errorf("feature policy CAS requires a non-zero expected revision and policy")
	}
	var updated *domain.CustomerResolutionFeaturePolicy
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		result := tx.Model(&persistence.CustomerResolutionFeaturePolicy{}).
			Where("id = ? AND revision = ?", 1, expectedRevision).
			Updates(map[string]any{
				"revision":                           expectedRevision + 1,
				"customer_resolution_writes_enabled": next.CustomerResolutionWritesEnabled,
				"candidate_scan_enabled":             next.CandidateScanEnabled,
				"merge_execution_enabled":            next.MergeExecutionEnabled,
				"split_execution_enabled":            next.SplitExecutionEnabled,
				"import_evidence_enabled":            next.ImportEvidenceEnabled,
				"carrier_registry_writes_enabled":    next.CarrierRegistryWritesEnabled,
				"actor_ref":                          strings.TrimSpace(next.ActorRef),
				"reason":                             strings.TrimSpace(next.Reason),
				"updated_at":                         now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errFeaturePolicyCASConflict
		}
		revision := persistence.CustomerResolutionFeaturePolicyRevision{
			Revision: expectedRevision + 1, CustomerResolutionWritesEnabled: next.CustomerResolutionWritesEnabled,
			CandidateScanEnabled: next.CandidateScanEnabled, MergeExecutionEnabled: next.MergeExecutionEnabled,
			SplitExecutionEnabled: next.SplitExecutionEnabled, ImportEvidenceEnabled: next.ImportEvidenceEnabled,
			CarrierRegistryWritesEnabled: next.CarrierRegistryWritesEnabled, ActorRef: strings.TrimSpace(next.ActorRef),
			Reason: strings.TrimSpace(next.Reason), CreatedAt: now,
		}
		if err := tx.Create(&revision).Error; err != nil {
			return err
		}
		copy := *next
		copy.Revision = expectedRevision + 1
		copy.ActorRef = revision.ActorRef
		copy.Reason = revision.Reason
		copy.UpdatedAt = now
		updated = &copy
		return nil
	})
	if errors.Is(err, errFeaturePolicyCASConflict) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("update customer resolution feature policy: %w", err)
	}
	return updated, true, nil
}

func featurePolicyToDomain(row *persistence.CustomerResolutionFeaturePolicy) *domain.CustomerResolutionFeaturePolicy {
	return &domain.CustomerResolutionFeaturePolicy{
		Revision: row.Revision, CustomerResolutionWritesEnabled: row.CustomerResolutionWritesEnabled,
		CandidateScanEnabled: row.CandidateScanEnabled, MergeExecutionEnabled: row.MergeExecutionEnabled,
		SplitExecutionEnabled: row.SplitExecutionEnabled, ImportEvidenceEnabled: row.ImportEvidenceEnabled,
		CarrierRegistryWritesEnabled: row.CarrierRegistryWritesEnabled, ActorRef: row.ActorRef,
		Reason: row.Reason, UpdatedAt: row.UpdatedAt,
	}
}

func featurePolicyUnavailable(err error) error {
	return &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeUnavailable,
		Detail: fmt.Sprintf("read customer resolution feature policy: %v", err)}
}

func featureEnabledForDB(ctx context.Context, db *gorm.DB, feature string) (bool, error) {
	return (&customerResolutionFeaturePolicyRepository{db: db}).FeatureEnabled(ctx, feature)
}

func requireFeatureForDB(ctx context.Context, db *gorm.DB, feature string) error {
	return (&customerResolutionFeaturePolicyRepository{db: db}).RequireFeature(ctx, feature)
}

func (r *profileRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *profileRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *addressRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *addressRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *mergeGovernanceRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *mergeGovernanceRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *mergeExecutionStore) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *mergeExecutionStore) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *splitExecutionStore) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *splitExecutionStore) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *importEvidenceRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *importEvidenceRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *externalCarrierRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *externalCarrierRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}

func (r *carrierMappingRepository) FeatureEnabled(ctx context.Context, feature string) (bool, error) {
	return featureEnabledForDB(ctx, r.db, feature)
}
func (r *carrierMappingRepository) RequireFeature(ctx context.Context, feature string) error {
	return requireFeatureForDB(ctx, r.db, feature)
}
