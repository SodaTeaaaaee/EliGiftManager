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
