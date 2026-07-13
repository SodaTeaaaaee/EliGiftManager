package dto

// UpdateWaveInput carries the wave rename/notes edit request (plan 3.2 / 5.2).
type UpdateWaveInput struct {
	WaveID    uint   `json:"waveId"`
	Name      string `json:"name"`
	Notes     string `json:"notes"`
	LevelTags string `json:"levelTags"`
}

// CloseWaveInput carries the explicit wave-closure request (plan 3.2 / 5.2).
// Force must be true, and Note non-empty, when the wave still has residual
// (unresolved) fulfillment items — this is the audit trail for a forced close.
type CloseWaveInput struct {
	WaveID uint   `json:"waveId"`
	Note   string `json:"note"`
	Force  bool   `json:"force"`
}

// CloseWaveResult reports the outcome of a CloseWave call, including whether the
// close was forced and how many residual items existed at close time.
type CloseWaveResult struct {
	Wave              WaveDTO `json:"wave"`
	Forced            bool    `json:"forced"`
	ResidualItemCount int     `json:"residualItemCount"`
}

// UnassignDemandInput carries the wave/demand pair for UnassignDemandFromWave.
type UnassignDemandInput struct {
	WaveID           uint `json:"waveId"`
	DemandDocumentID uint `json:"demandDocumentId"`
}

// BatchAssignDemandInput carries a batch of demand documents to assign to one wave.
type BatchAssignDemandInput struct {
	WaveID uint   `json:"waveId"`
	DocIDs []uint `json:"docIds"`
}

// BatchAssignDemandItemResult reports the per-item outcome of a batch assignment.
type BatchAssignDemandItemResult struct {
	DemandDocumentID uint   `json:"demandDocumentId"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
}

// BatchAssignDemandResult aggregates per-item results for BatchAssignDemandToWave,
// implementing partial-success semantics — one failing item does not abort the rest.
type BatchAssignDemandResult struct {
	Results      []BatchAssignDemandItemResult `json:"results"`
	SuccessCount int                           `json:"successCount"`
	FailureCount int                           `json:"failureCount"`
}
