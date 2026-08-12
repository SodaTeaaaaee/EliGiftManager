package dto

import "time"

type CustomerResolutionFeaturePolicyDTO struct {
	Revision                        uint64    `json:"revision"`
	CustomerResolutionWritesEnabled bool      `json:"customerResolutionWritesEnabled"`
	CandidateScanEnabled            bool      `json:"candidateScanEnabled"`
	MergeExecutionEnabled           bool      `json:"mergeExecutionEnabled"`
	SplitExecutionEnabled           bool      `json:"splitExecutionEnabled"`
	ImportEvidenceEnabled           bool      `json:"importEvidenceEnabled"`
	CarrierRegistryWritesEnabled    bool      `json:"carrierRegistryWritesEnabled"`
	ActorRef                        string    `json:"actorRef"`
	Reason                          string    `json:"reason"`
	UpdatedAt                       time.Time `json:"updatedAt" ts_type:"string"`
}

type UpdateCustomerResolutionFeaturePolicyInput struct {
	ExpectedRevision                uint64 `json:"expectedRevision"`
	CustomerResolutionWritesEnabled bool   `json:"customerResolutionWritesEnabled"`
	CandidateScanEnabled            bool   `json:"candidateScanEnabled"`
	MergeExecutionEnabled           bool   `json:"mergeExecutionEnabled"`
	SplitExecutionEnabled           bool   `json:"splitExecutionEnabled"`
	ImportEvidenceEnabled           bool   `json:"importEvidenceEnabled"`
	CarrierRegistryWritesEnabled    bool   `json:"carrierRegistryWritesEnabled"`
	ActorRef                        string `json:"actorRef"`
	Reason                          string `json:"reason"`
}
