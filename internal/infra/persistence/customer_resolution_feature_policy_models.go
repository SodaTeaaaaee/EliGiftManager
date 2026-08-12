package persistence

import "time"

type CustomerResolutionFeaturePolicy struct {
	ID                              uint   `gorm:"primaryKey"`
	Revision                        uint64 `gorm:"not null"`
	CustomerResolutionWritesEnabled bool   `gorm:"not null"`
	CandidateScanEnabled            bool   `gorm:"not null"`
	MergeExecutionEnabled           bool   `gorm:"not null"`
	SplitExecutionEnabled           bool   `gorm:"not null"`
	ImportEvidenceEnabled           bool   `gorm:"not null"`
	CarrierRegistryWritesEnabled    bool   `gorm:"not null"`
	ActorRef                        string `gorm:"type:text;not null;default:''"`
	Reason                          string `gorm:"type:text;not null;default:''"`
	UpdatedAt                       time.Time
}

func (CustomerResolutionFeaturePolicy) TableName() string {
	return "customer_resolution_feature_policy"
}

type CustomerResolutionFeaturePolicyRevision struct {
	ID                              uint   `gorm:"primaryKey;autoIncrement"`
	Revision                        uint64 `gorm:"not null;uniqueIndex"`
	CustomerResolutionWritesEnabled bool   `gorm:"not null"`
	CandidateScanEnabled            bool   `gorm:"not null"`
	MergeExecutionEnabled           bool   `gorm:"not null"`
	SplitExecutionEnabled           bool   `gorm:"not null"`
	ImportEvidenceEnabled           bool   `gorm:"not null"`
	CarrierRegistryWritesEnabled    bool   `gorm:"not null"`
	ActorRef                        string `gorm:"type:text;not null;default:''"`
	Reason                          string `gorm:"type:text;not null;default:''"`
	CreatedAt                       time.Time
}

func (CustomerResolutionFeaturePolicyRevision) TableName() string {
	return "customer_resolution_feature_policy_revisions"
}
