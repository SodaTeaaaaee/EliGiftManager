package domain

import (
	"context"
	"fmt"
	"time"
)

const (
	CustomerResolutionFeatureWrites          = "customerResolutionWritesEnabled"
	CustomerResolutionFeatureCandidateScan   = "candidateScanEnabled"
	CustomerResolutionFeatureMergeExecution  = "mergeExecutionEnabled"
	CustomerResolutionFeatureSplitExecution  = "splitExecutionEnabled"
	CustomerResolutionFeatureImportEvidence  = "importEvidenceEnabled"
	CustomerResolutionFeatureCarrierRegistry = "carrierRegistryWritesEnabled"

	FeaturePolicyCodeWritesDisabled          = "customer_resolution_writes_disabled"
	FeaturePolicyCodeCandidateScanDisabled   = "candidate_scan_disabled"
	FeaturePolicyCodeMergeExecutionDisabled  = "merge_execution_disabled"
	FeaturePolicyCodeSplitExecutionDisabled  = "split_execution_disabled"
	FeaturePolicyCodeImportEvidenceDisabled  = "import_evidence_disabled"
	FeaturePolicyCodeCarrierRegistryDisabled = "carrier_registry_writes_disabled"
	FeaturePolicyCodeRevisionConflict        = "customer_resolution_feature_policy_revision_conflict"
	FeaturePolicyCodeUnavailable             = "customer_resolution_feature_policy_unavailable"
)

type CustomerResolutionFeaturePolicy struct {
	Revision                        uint64
	CustomerResolutionWritesEnabled bool
	CandidateScanEnabled            bool
	MergeExecutionEnabled           bool
	SplitExecutionEnabled           bool
	ImportEvidenceEnabled           bool
	CarrierRegistryWritesEnabled    bool
	ActorRef                        string
	Reason                          string
	UpdatedAt                       time.Time
}

func DefaultCustomerResolutionFeaturePolicy() CustomerResolutionFeaturePolicy {
	return CustomerResolutionFeaturePolicy{
		Revision: 1, CustomerResolutionWritesEnabled: true, CandidateScanEnabled: true,
		MergeExecutionEnabled: true, SplitExecutionEnabled: true, ImportEvidenceEnabled: true,
		CarrierRegistryWritesEnabled: true,
	}
}

type FeaturePolicyError struct {
	Code    string
	Feature string
	Detail  string
}

func (e *FeaturePolicyError) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func FeaturePolicyDisabledError(feature string) error {
	code := FeaturePolicyCodeUnavailable
	switch feature {
	case CustomerResolutionFeatureWrites:
		code = FeaturePolicyCodeWritesDisabled
	case CustomerResolutionFeatureCandidateScan:
		code = FeaturePolicyCodeCandidateScanDisabled
	case CustomerResolutionFeatureMergeExecution:
		code = FeaturePolicyCodeMergeExecutionDisabled
	case CustomerResolutionFeatureSplitExecution:
		code = FeaturePolicyCodeSplitExecutionDisabled
	case CustomerResolutionFeatureImportEvidence:
		code = FeaturePolicyCodeImportEvidenceDisabled
	case CustomerResolutionFeatureCarrierRegistry:
		code = FeaturePolicyCodeCarrierRegistryDisabled
	}
	return &FeaturePolicyError{Code: code, Feature: feature, Detail: "feature is disabled by customer resolution policy"}
}

type CustomerResolutionFeatureGate interface {
	FeatureEnabled(ctx context.Context, feature string) (bool, error)
	RequireFeature(ctx context.Context, feature string) error
}

type CustomerResolutionFeaturePolicyRepository interface {
	CustomerResolutionFeatureGate
	GetFeaturePolicy(ctx context.Context) (*CustomerResolutionFeaturePolicy, error)
	UpdateFeaturePolicyCAS(ctx context.Context, expectedRevision uint64, next *CustomerResolutionFeaturePolicy) (*CustomerResolutionFeaturePolicy, bool, error)
}
