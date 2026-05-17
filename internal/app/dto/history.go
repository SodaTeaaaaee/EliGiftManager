package dto

type HistoryNodeDTO struct {
	ID                   uint   `json:"id"`
	ParentNodeID         uint   `json:"parentNodeId"`
	PreferredRedoChildID uint   `json:"preferredRedoChildId"`
	CommandKind          string `json:"commandKind"`
	CommandSummary       string `json:"commandSummary"`
	ProjectionHash       string `json:"projectionHash"`
	CheckpointHint       bool   `json:"checkpointHint"`
	CreatedAt            string `json:"createdAt"`
	CreatedBy            string `json:"createdBy"`
}

// HistoryGraphDTO represents the full history tree for a scope.
type HistoryGraphDTO struct {
	ScopeID       uint                  `json:"scopeId"`
	CurrentHeadID uint                  `json:"currentHeadId"`
	Nodes         []HistoryGraphNodeDTO `json:"nodes"`
}

// HistoryGraphNodeDTO is a node in the history graph with relationship info.
type HistoryGraphNodeDTO struct {
	ID                   uint   `json:"id"`
	ParentNodeID         uint   `json:"parentNodeId"`
	PreferredRedoChildID uint   `json:"preferredRedoChildId"`
	CommandKind          string `json:"commandKind"`
	CommandSummary       string `json:"commandSummary"`
	ProjectionHash       string `json:"projectionHash"`
	CheckpointHint       bool   `json:"checkpointHint"`
	CreatedAt            string `json:"createdAt"`
	CreatedBy            string `json:"createdBy"`
	IsCurrentHead        bool   `json:"isCurrentHead"`
	IsPinned             bool   `json:"isPinned"`
	ChildCount           int    `json:"childCount"`
}

// HistoryNodeDetailDTO extends HistoryNodeDTO with pins and checkpoint info.
type HistoryNodeDetailDTO struct {
	HistoryNodeDTO
	Pins          []HistoryPinDTO `json:"pins"`
	HasCheckpoint bool            `json:"hasCheckpoint"`
}

// HistoryPinDTO represents a history pin.
type HistoryPinDTO struct {
	ID            uint   `json:"id"`
	HistoryNodeID uint   `json:"historyNodeId"`
	PinKind       string `json:"pinKind"`
	RefType       string `json:"refType"`
	RefID         uint   `json:"refId"`
	CreatedAt     string `json:"createdAt"`
}
