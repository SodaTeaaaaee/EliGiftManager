package persistence

import "github.com/SodaTeaaaaee/EliGiftManager/internal/domain"

func CustomerNameObservationFromDomain(d *domain.CustomerNameObservation) *CustomerNameObservation {
	return &CustomerNameObservation{
		CustomerProfileID: d.CustomerProfileID, Name: d.Name, NormalizedName: d.NormalizedName,
		SourceEventKey: d.SourceEventKey, EpisodeKey: d.EpisodeKey, ObservationCount: d.ObservationCount,
		NameKind: d.NameKind, Authority: d.Authority, TrustScore: d.TrustScore,
		SourceIntegrationProfileID: d.SourceIntegrationProfileID, SourceDocumentID: d.SourceDocumentID,
		SourceIdentityID: d.SourceIdentityID, ObservedAt: d.ObservedAt, FirstSeenAt: d.FirstSeenAt,
		LastSeenAt: d.LastSeenAt, IsPinned: d.IsPinned, IsActive: d.IsActive, ExtraData: d.ExtraData,
	}
}

func CustomerNameObservationToDomain(p *CustomerNameObservation) *domain.CustomerNameObservation {
	return &domain.CustomerNameObservation{
		ID: p.ID, CustomerProfileID: p.CustomerProfileID, OriginProfileID: p.CustomerProfileID, Name: p.Name, NormalizedName: p.NormalizedName,
		SourceEventKey: p.SourceEventKey, EpisodeKey: p.EpisodeKey, ObservationCount: p.ObservationCount,
		NameKind: p.NameKind, Authority: p.Authority, TrustScore: p.TrustScore,
		SourceIntegrationProfileID: p.SourceIntegrationProfileID, SourceDocumentID: p.SourceDocumentID,
		SourceIdentityID: p.SourceIdentityID, ObservedAt: p.ObservedAt, FirstSeenAt: p.FirstSeenAt,
		LastSeenAt: p.LastSeenAt, IsPinned: p.IsPinned, IsActive: p.IsActive, ExtraData: p.ExtraData,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func CustomerNameEventFromDomain(d *domain.CustomerNameEvent) *CustomerNameEvent {
	return &CustomerNameEvent{ID: d.ID, EventKey: d.EventKey, CustomerProfileID: d.CustomerProfileID, ObservationID: d.ObservationID,
		EventKind: d.EventKind, PreviousName: d.PreviousName, NewName: d.NewName, ReasonCode: d.ReasonCode,
		ActorRef: d.ActorRef, Payload: d.Payload, CreatedAt: d.CreatedAt}
}

func CustomerNameEventToDomain(p *CustomerNameEvent) *domain.CustomerNameEvent {
	return &domain.CustomerNameEvent{ID: p.ID, EventKey: p.EventKey, CustomerProfileID: p.CustomerProfileID, ObservationID: p.ObservationID,
		EventKind: p.EventKind, PreviousName: p.PreviousName, NewName: p.NewName, ReasonCode: p.ReasonCode,
		ActorRef: p.ActorRef, Payload: p.Payload, CreatedAt: p.CreatedAt}
}

func CustomerProfileOriginFromDomain(d *domain.CustomerProfileOrigin) *CustomerProfileOrigin {
	return &CustomerProfileOrigin{CustomerProfileID: d.CustomerProfileID, OriginKind: d.OriginKind,
		SourceIntegrationProfileID: d.SourceIntegrationProfileID, SourceDocumentID: d.SourceDocumentID,
		ExternalRef: d.ExternalRef, IsProvisional: d.IsProvisional, FirstSeenAt: d.FirstSeenAt,
		LastSeenAt: d.LastSeenAt, ExtraData: d.ExtraData}
}

func CustomerProfileOriginToDomain(p *CustomerProfileOrigin) *domain.CustomerProfileOrigin {
	return &domain.CustomerProfileOrigin{ID: p.ID, CustomerProfileID: p.CustomerProfileID, OriginKind: p.OriginKind,
		SourceIntegrationProfileID: p.SourceIntegrationProfileID, SourceDocumentID: p.SourceDocumentID,
		ExternalRef: p.ExternalRef, IsProvisional: p.IsProvisional, FirstSeenAt: p.FirstSeenAt,
		LastSeenAt: p.LastSeenAt, ExtraData: p.ExtraData, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func MergeCandidateFromDomain(d *domain.MergeCandidate) *MergeCandidate {
	return &MergeCandidate{SourceProfileID: d.SourceProfileID, TargetProfileID: d.TargetProfileID,
		Status: d.Status, Score: d.Score, MergePolicyRevisionID: d.MergePolicyRevisionID, Reason: d.Reason,
		CanonicalPairKey: d.CanonicalPairKey, EvidenceHash: d.EvidenceHash, PolicyVersion: d.PolicyVersion,
		ExplanationCode: d.ExplanationCode, Confidence: d.Confidence, Blockers: d.Blockers,
		LastEvaluatedAt: d.LastEvaluatedAt, ExpiresAt: d.ExpiresAt, ScanRunID: d.ScanRunID,
		RowVersion: d.RowVersion, ExecutedMergeRecordID: d.ExecutedMergeRecordID, ExecutedAt: d.ExecutedAt}
}

func MergeCandidateToDomain(p *MergeCandidate) *domain.MergeCandidate {
	return &domain.MergeCandidate{ID: p.ID, SourceProfileID: p.SourceProfileID, TargetProfileID: p.TargetProfileID,
		Status: p.Status, Score: p.Score, MergePolicyRevisionID: p.MergePolicyRevisionID, Reason: p.Reason,
		CanonicalPairKey: p.CanonicalPairKey, EvidenceHash: p.EvidenceHash, PolicyVersion: p.PolicyVersion,
		ExplanationCode: p.ExplanationCode, Confidence: p.Confidence, Blockers: p.Blockers,
		LastEvaluatedAt: p.LastEvaluatedAt, ExpiresAt: p.ExpiresAt, ScanRunID: p.ScanRunID,
		RowVersion: p.RowVersion, ExecutedMergeRecordID: p.ExecutedMergeRecordID, ExecutedAt: p.ExecutedAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func MergeEvidenceFromDomain(d *domain.MergeEvidence) *MergeEvidence {
	return &MergeEvidence{ID: d.ID, MergeCandidateID: d.MergeCandidateID, EvidenceKind: d.EvidenceKind,
		SourceRef: d.SourceRef, Weight: d.Weight, Confidence: d.Confidence, Payload: d.Payload,
		EvidenceKey: d.EvidenceKey, Polarity: d.Polarity, ExplanationCode: d.ExplanationCode,
		ValueHash: d.ValueHash, MaskedValue: d.MaskedValue, SourceEntityType: d.SourceEntityType,
		SourceEntityID: d.SourceEntityID, ObservedAt: d.ObservedAt, CreatedAt: d.CreatedAt}
}

func MergeEvidenceToDomain(p *MergeEvidence) *domain.MergeEvidence {
	return &domain.MergeEvidence{ID: p.ID, MergeCandidateID: p.MergeCandidateID, EvidenceKind: p.EvidenceKind,
		SourceRef: p.SourceRef, Weight: p.Weight, Confidence: p.Confidence, Payload: p.Payload,
		EvidenceKey: p.EvidenceKey, Polarity: p.Polarity, ExplanationCode: p.ExplanationCode,
		ValueHash: p.ValueHash, MaskedValue: p.MaskedValue, SourceEntityType: p.SourceEntityType,
		SourceEntityID: p.SourceEntityID, ObservedAt: p.ObservedAt, CreatedAt: p.CreatedAt}
}

func MergePolicyFromDomain(d *domain.MergePolicy) *MergePolicy {
	return &MergePolicy{PolicyKey: d.PolicyKey, Name: d.Name, IsActive: d.IsActive,
		DefaultAction: d.DefaultAction, CurrentRevisionID: d.CurrentRevisionID, ExtraData: d.ExtraData,
		RowVersion: d.RowVersion, NeedsScan: d.NeedsScan, LastScanAt: d.LastScanAt}
}

func MergePolicyToDomain(p *MergePolicy) *domain.MergePolicy {
	return &domain.MergePolicy{ID: p.ID, PolicyKey: p.PolicyKey, Name: p.Name, IsActive: p.IsActive,
		DefaultAction: p.DefaultAction, CurrentRevisionID: p.CurrentRevisionID, ExtraData: p.ExtraData,
		RowVersion: p.RowVersion, NeedsScan: p.NeedsScan, LastScanAt: p.LastScanAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func MergePolicyRevisionFromDomain(d *domain.MergePolicyRevision) *MergePolicyRevision {
	return &MergePolicyRevision{ID: d.ID, MergePolicyID: d.MergePolicyID, Revision: d.Revision,
		Action: d.Action, Rules: d.Rules, Checksum: d.Checksum, CreatedBy: d.CreatedBy,
		SchemaVersion: d.SchemaVersion, CreatedAt: d.CreatedAt}
}

func MergePolicyRevisionToDomain(p *MergePolicyRevision) *domain.MergePolicyRevision {
	return &domain.MergePolicyRevision{ID: p.ID, MergePolicyID: p.MergePolicyID, Revision: p.Revision,
		Action: p.Action, Rules: p.Rules, Checksum: p.Checksum, CreatedBy: p.CreatedBy,
		SchemaVersion: p.SchemaVersion, CreatedAt: p.CreatedAt}
}

func MergeScanRunFromDomain(d *domain.MergeScanRun) *MergeScanRun {
	return &MergeScanRun{ID: d.ID, MergePolicyID: d.MergePolicyID, PolicyRevisionID: d.PolicyRevisionID,
		PolicyVersion: d.PolicyVersion, Status: d.Status, StartedAt: d.StartedAt, CompletedAt: d.CompletedAt,
		ProfilesScanned: d.ProfilesScanned, PairsEvaluated: d.PairsEvaluated,
		CandidatesCreated: d.CandidatesCreated, CandidatesUpdated: d.CandidatesUpdated,
		CandidatesBlocked: d.CandidatesBlocked, ErrorMessage: d.ErrorMessage,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

func MergeScanRunToDomain(p *MergeScanRun) *domain.MergeScanRun {
	return &domain.MergeScanRun{ID: p.ID, MergePolicyID: p.MergePolicyID, PolicyRevisionID: p.PolicyRevisionID,
		PolicyVersion: p.PolicyVersion, Status: p.Status, StartedAt: p.StartedAt, CompletedAt: p.CompletedAt,
		ProfilesScanned: p.ProfilesScanned, PairsEvaluated: p.PairsEvaluated,
		CandidatesCreated: p.CandidatesCreated, CandidatesUpdated: p.CandidatesUpdated,
		CandidatesBlocked: p.CandidatesBlocked, ErrorMessage: p.ErrorMessage,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func MergeMovedEntityFromDomain(d *domain.MergeMovedEntity) *MergeMovedEntity {
	return &MergeMovedEntity{ID: d.ID, MergeRecordID: d.MergeRecordID, EntityType: d.EntityType, EntityID: d.EntityID,
		FromProfileID: d.FromProfileID, ToProfileID: d.ToProfileID, MoveOrder: d.MoveOrder,
		BeforeSnapshot: d.BeforeSnapshot, MutationKind: d.MutationKind, RestoreMode: d.RestoreMode,
		SnapshotVersion: d.SnapshotVersion, AfterSnapshot: d.AfterSnapshot, AfterStateHash: d.AfterStateHash,
		EntityUpdatedAtAfter: d.EntityUpdatedAtAfter, UndoState: d.UndoState, UndoBlockerCode: d.UndoBlockerCode,
		RevertOperationKey: d.RevertOperationKey, RevertedAt: d.RevertedAt, CreatedAt: d.CreatedAt}
}

func MergeMovedEntityToDomain(p *MergeMovedEntity) *domain.MergeMovedEntity {
	return &domain.MergeMovedEntity{ID: p.ID, MergeRecordID: p.MergeRecordID, EntityType: p.EntityType, EntityID: p.EntityID,
		FromProfileID: p.FromProfileID, ToProfileID: p.ToProfileID, MoveOrder: p.MoveOrder,
		BeforeSnapshot: p.BeforeSnapshot, MutationKind: p.MutationKind, RestoreMode: p.RestoreMode,
		SnapshotVersion: p.SnapshotVersion, AfterSnapshot: p.AfterSnapshot, AfterStateHash: p.AfterStateHash,
		EntityUpdatedAtAfter: p.EntityUpdatedAtAfter, UndoState: p.UndoState, UndoBlockerCode: p.UndoBlockerCode,
		RevertOperationKey: p.RevertOperationKey, RevertedAt: p.RevertedAt, CreatedAt: p.CreatedAt}
}
