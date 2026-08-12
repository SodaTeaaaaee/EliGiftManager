package persistence

import "github.com/SodaTeaaaaee/EliGiftManager/internal/domain"

func CustomerSplitRecordFromDomain(value *domain.CustomerSplitRecord) *CustomerSplitRecord {
	return &CustomerSplitRecord{
		ID: value.ID, OperationKey: value.OperationKey, CommandHash: value.CommandHash,
		PreviewHash: value.PreviewHash, MovePlanHash: value.MovePlanHash, Status: value.Status,
		SourceProfileID: value.SourceProfileID, TargetProfileID: value.TargetProfileID,
		TargetStrategy: value.TargetStrategy, ActorRef: value.ActorRef, DecisionReason: value.DecisionReason,
		SourceRowVersion: value.SourceRowVersion, TargetRowVersion: value.TargetRowVersion,
		SourceRowVersionAfter: value.SourceRowVersionAfter, TargetRowVersionAfter: value.TargetRowVersionAfter,
		SourceProfileSnapshot: value.SourceProfileSnapshot, TargetProfileSnapshot: value.TargetProfileSnapshot,
		Payload: value.Payload, RowVersion: value.RowVersion, ReverseOperationKind: value.ReverseOperationKind,
		CreatedAt: value.CreatedAt, CompletedAt: value.CompletedAt,
	}
}

func CustomerSplitRecordToDomain(value *CustomerSplitRecord) *domain.CustomerSplitRecord {
	return &domain.CustomerSplitRecord{
		ID: value.ID, OperationKey: value.OperationKey, CommandHash: value.CommandHash,
		PreviewHash: value.PreviewHash, MovePlanHash: value.MovePlanHash, Status: value.Status,
		SourceProfileID: value.SourceProfileID, TargetProfileID: value.TargetProfileID,
		TargetStrategy: value.TargetStrategy, ActorRef: value.ActorRef, DecisionReason: value.DecisionReason,
		SourceRowVersion: value.SourceRowVersion, TargetRowVersion: value.TargetRowVersion,
		SourceRowVersionAfter: value.SourceRowVersionAfter, TargetRowVersionAfter: value.TargetRowVersionAfter,
		SourceProfileSnapshot: value.SourceProfileSnapshot, TargetProfileSnapshot: value.TargetProfileSnapshot,
		Payload: value.Payload, RowVersion: value.RowVersion, ReverseOperationKind: value.ReverseOperationKind,
		CreatedAt: value.CreatedAt, CompletedAt: value.CompletedAt,
	}
}

func SplitMovedEntityFromDomain(value *domain.SplitMovedEntity) *SplitMovedEntity {
	return &SplitMovedEntity{
		ID: value.ID, SplitRecordID: value.SplitRecordID, EntityType: value.EntityType,
		EntityID: value.EntityID, FromProfileID: value.FromProfileID, ToProfileID: value.ToProfileID,
		MoveOrder: value.MoveOrder, BeforeSnapshot: value.BeforeSnapshot, AfterSnapshot: value.AfterSnapshot,
		AfterStateHash: value.AfterStateHash, MutationKind: value.MutationKind,
		SnapshotVersion: value.SnapshotVersion, CreatedAt: value.CreatedAt,
	}
}

func SplitMovedEntityToDomain(value *SplitMovedEntity) *domain.SplitMovedEntity {
	return &domain.SplitMovedEntity{
		ID: value.ID, SplitRecordID: value.SplitRecordID, EntityType: value.EntityType,
		EntityID: value.EntityID, FromProfileID: value.FromProfileID, ToProfileID: value.ToProfileID,
		MoveOrder: value.MoveOrder, BeforeSnapshot: value.BeforeSnapshot, AfterSnapshot: value.AfterSnapshot,
		AfterStateHash: value.AfterStateHash, MutationKind: value.MutationKind,
		SnapshotVersion: value.SnapshotVersion, CreatedAt: value.CreatedAt,
	}
}
