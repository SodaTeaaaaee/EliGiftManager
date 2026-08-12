package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type CustomerResolutionFeaturePolicyUseCase struct {
	repo domain.CustomerResolutionFeaturePolicyRepository
}

func NewCustomerResolutionFeaturePolicyUseCase(repo domain.CustomerResolutionFeaturePolicyRepository) *CustomerResolutionFeaturePolicyUseCase {
	return &CustomerResolutionFeaturePolicyUseCase{repo: repo}
}

func (uc *CustomerResolutionFeaturePolicyUseCase) Get(ctx context.Context) (*dto.CustomerResolutionFeaturePolicyDTO, error) {
	policy, err := uc.repo.GetFeaturePolicy(ctx)
	if err != nil {
		return nil, err
	}
	result := customerResolutionFeaturePolicyDTO(policy)
	return &result, nil
}

func (uc *CustomerResolutionFeaturePolicyUseCase) Update(
	ctx context.Context,
	input dto.UpdateCustomerResolutionFeaturePolicyInput,
) (*dto.CustomerResolutionFeaturePolicyDTO, error) {
	if input.ExpectedRevision == 0 {
		return nil, &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeRevisionConflict,
			Detail: "expectedRevision must be non-zero"}
	}
	next := &domain.CustomerResolutionFeaturePolicy{
		CustomerResolutionWritesEnabled: input.CustomerResolutionWritesEnabled,
		CandidateScanEnabled:            input.CandidateScanEnabled, MergeExecutionEnabled: input.MergeExecutionEnabled,
		SplitExecutionEnabled: input.SplitExecutionEnabled, ImportEvidenceEnabled: input.ImportEvidenceEnabled,
		CarrierRegistryWritesEnabled: input.CarrierRegistryWritesEnabled, ActorRef: strings.TrimSpace(input.ActorRef),
		Reason: strings.TrimSpace(input.Reason),
	}
	updated, applied, err := uc.repo.UpdateFeaturePolicyCAS(ctx, input.ExpectedRevision, next)
	if err != nil {
		return nil, err
	}
	if !applied {
		return nil, &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeRevisionConflict,
			Detail: fmt.Sprintf("expected revision %d is stale", input.ExpectedRevision)}
	}
	result := customerResolutionFeaturePolicyDTO(updated)
	return &result, nil
}

func customerResolutionFeaturePolicyDTO(policy *domain.CustomerResolutionFeaturePolicy) dto.CustomerResolutionFeaturePolicyDTO {
	return dto.CustomerResolutionFeaturePolicyDTO{
		Revision: policy.Revision, CustomerResolutionWritesEnabled: policy.CustomerResolutionWritesEnabled,
		CandidateScanEnabled: policy.CandidateScanEnabled, MergeExecutionEnabled: policy.MergeExecutionEnabled,
		SplitExecutionEnabled: policy.SplitExecutionEnabled, ImportEvidenceEnabled: policy.ImportEvidenceEnabled,
		CarrierRegistryWritesEnabled: policy.CarrierRegistryWritesEnabled, ActorRef: policy.ActorRef,
		Reason: policy.Reason, UpdatedAt: policy.UpdatedAt,
	}
}

func featureGateFrom(source any) (domain.CustomerResolutionFeatureGate, error) {
	gate, ok := source.(domain.CustomerResolutionFeatureGate)
	if !ok || gate == nil {
		return nil, &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeUnavailable,
			Detail: "customer resolution feature gate is not configured"}
	}
	return gate, nil
}

func requireCustomerResolutionFeature(ctx context.Context, source any, feature string) error {
	gate, err := featureGateFrom(source)
	if err != nil {
		return err
	}
	return gate.RequireFeature(ctx, feature)
}

func customerResolutionFeatureEnabled(ctx context.Context, source any, feature string) (bool, error) {
	gate, err := featureGateFrom(source)
	if err != nil {
		return false, err
	}
	return gate.FeatureEnabled(ctx, feature)
}
