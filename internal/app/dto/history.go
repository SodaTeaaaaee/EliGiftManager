package dto

type HistoryNodeDTO struct {
	ID             uint   `json:"id"`
	CommandKind    string `json:"commandKind"`
	CommandSummary string `json:"commandSummary"`
	CreatedAt      string `json:"createdAt"`
	CreatedBy      string `json:"createdBy"`
}
