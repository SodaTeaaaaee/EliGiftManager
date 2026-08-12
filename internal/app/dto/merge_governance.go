package dto

import "time"

type MergePolicyRulesDTO struct {
	CandidateDetectionEnabled bool   `json:"candidateDetectionEnabled"`
	EmailEvidenceMode         string `json:"emailEvidenceMode"`
	PhoneEvidenceMode         string `json:"phoneEvidenceMode"`
	ExecutionMode             string `json:"executionMode"`
}

type MergePolicyDTO struct {
	ID           uint                `json:"id"`
	PolicyKey    string              `json:"policyKey"`
	Revision     uint                `json:"revision"`
	Rules        MergePolicyRulesDTO `json:"rules"`
	NeedsScan    bool                `json:"needsScan"`
	LastScanAt   *time.Time          `json:"lastScanAt" ts_type:"string"`
	RevisionTime time.Time           `json:"revisionTime" ts_type:"string"`
}

type UpdateMergePolicyInput struct {
	ExpectedRevision uint                `json:"expectedRevision"`
	Rules            MergePolicyRulesDTO `json:"rules"`
	ActorRef         string              `json:"actorRef"`
}

type MergeEvidenceDTO struct {
	ID              uint       `json:"id"`
	EvidenceKind    string     `json:"evidenceKind"`
	Polarity        string     `json:"polarity"`
	ExplanationCode string     `json:"explanationCode"`
	Confidence      float64    `json:"confidence"`
	ValueHash       string     `json:"valueHash"`
	MaskedValue     string     `json:"maskedValue"`
	ObservedAt      *time.Time `json:"observedAt" ts_type:"string"`
}

type MergeCandidateDTO struct {
	ID               uint               `json:"id"`
	SourceProfileID  uint               `json:"sourceProfileId"`
	TargetProfileID  uint               `json:"targetProfileId"`
	Status           string             `json:"status"`
	Confidence       float64            `json:"confidence"`
	ExplanationCode  string             `json:"explanationCode"`
	EvidenceHash     string             `json:"evidenceHash"`
	PolicyVersion    uint               `json:"policyVersion"`
	PolicyRevisionID *uint              `json:"policyRevisionId"`
	BlockerCodes     []string           `json:"blockerCodes"`
	LastEvaluatedAt  *time.Time         `json:"lastEvaluatedAt" ts_type:"string"`
	ExpiresAt        *time.Time         `json:"expiresAt" ts_type:"string"`
	Evidence         []MergeEvidenceDTO `json:"evidence"`
	CreatedAt        time.Time          `json:"createdAt" ts_type:"string"`
	UpdatedAt        time.Time          `json:"updatedAt" ts_type:"string"`
	RowVersion       uint64             `json:"rowVersion"`
}

type DismissMergeCandidateInput struct {
	ID            uint   `json:"id"`
	EvidenceHash  string `json:"evidenceHash"`
	PolicyVersion uint   `json:"policyVersion"`
}

type MergeScanRunDTO struct {
	ID                uint       `json:"id"`
	PolicyVersion     uint       `json:"policyVersion"`
	Status            string     `json:"status"`
	StartedAt         time.Time  `json:"startedAt" ts_type:"string"`
	CompletedAt       *time.Time `json:"completedAt" ts_type:"string"`
	ProfilesScanned   uint       `json:"profilesScanned"`
	PairsEvaluated    uint       `json:"pairsEvaluated"`
	CandidatesCreated uint       `json:"candidatesCreated"`
	CandidatesUpdated uint       `json:"candidatesUpdated"`
	CandidatesBlocked uint       `json:"candidatesBlocked"`
	ErrorMessage      string     `json:"errorMessage"`
}
