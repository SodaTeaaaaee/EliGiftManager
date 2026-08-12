package persistence

import "time"

type CustomerSplitRecord struct {
	ID                    uint   `gorm:"primaryKey;autoIncrement"`
	OperationKey          string `gorm:"type:text;not null;default:''"`
	CommandHash           string `gorm:"type:text;not null;default:''"`
	PreviewHash           string `gorm:"type:text;not null;default:''"`
	MovePlanHash          string `gorm:"type:text;not null;default:''"`
	Status                string `gorm:"type:text;not null;default:'executing'"`
	SourceProfileID       uint   `gorm:"not null"`
	TargetProfileID       uint   `gorm:"not null"`
	TargetStrategy        string `gorm:"type:text;not null;default:'create_new'"`
	ActorRef              string `gorm:"type:text;not null;default:''"`
	DecisionReason        string `gorm:"type:text;not null;default:''"`
	SourceRowVersion      uint64 `gorm:"not null;default:0"`
	TargetRowVersion      uint64 `gorm:"not null;default:0"`
	SourceRowVersionAfter uint64 `gorm:"not null;default:0"`
	TargetRowVersionAfter uint64 `gorm:"not null;default:0"`
	SourceProfileSnapshot string `gorm:"type:text;not null;default:''"`
	TargetProfileSnapshot string `gorm:"type:text;not null;default:''"`
	Payload               string `gorm:"type:text;not null;default:''"`
	RowVersion            uint64 `gorm:"not null;default:1"`
	ReverseOperationKind  string `gorm:"type:text;not null;default:'manual_merge_required'"`
	CreatedAt             time.Time
	CompletedAt           *time.Time
}

func (CustomerSplitRecord) TableName() string { return "customer_split_records" }

type SplitMovedEntity struct {
	ID              uint   `gorm:"primaryKey;autoIncrement"`
	SplitRecordID   uint   `gorm:"not null"`
	EntityType      string `gorm:"type:text;not null"`
	EntityID        uint   `gorm:"not null"`
	FromProfileID   uint   `gorm:"not null"`
	ToProfileID     uint   `gorm:"not null"`
	MoveOrder       uint   `gorm:"not null;default:0"`
	BeforeSnapshot  string `gorm:"type:text;not null;default:''"`
	AfterSnapshot   string `gorm:"type:text;not null;default:''"`
	AfterStateHash  string `gorm:"type:text;not null;default:''"`
	MutationKind    string `gorm:"type:text;not null;default:''"`
	SnapshotVersion uint   `gorm:"not null;default:1"`
	CreatedAt       time.Time
}

func (SplitMovedEntity) TableName() string { return "split_moved_entities" }

type CustomerSplitOperationEvent struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	SplitRecordID uint   `gorm:"not null"`
	EventKey      string `gorm:"type:text;not null;default:''"`
	OperationKey  string `gorm:"type:text;not null;default:''"`
	EventType     string `gorm:"type:text;not null"`
	Status        string `gorm:"type:text;not null"`
	ActorRef      string `gorm:"type:text;not null;default:''"`
	ReasonCode    string `gorm:"type:text;not null;default:''"`
	Payload       string `gorm:"type:text;not null;default:''"`
	CreatedAt     time.Time
}

func (CustomerSplitOperationEvent) TableName() string {
	return "customer_split_operation_events"
}
