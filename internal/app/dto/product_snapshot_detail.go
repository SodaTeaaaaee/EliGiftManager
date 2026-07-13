package dto

// SnapshotProductDetailItem describes the outcome of snapshotting a single
// product master into a wave: either a new wave-scoped Product row was
// created, or one already existed for that wave/platform/SKU and the
// existing row was returned unchanged (skip-detail, plan 5.3).
type SnapshotProductDetailItem struct {
	MasterID       uint       `json:"masterId"`
	Product        ProductDTO `json:"product"`
	AlreadyExisted bool       `json:"alreadyExisted"`
}

// SnapshotProductsDetailedResult is the per-master detail result of a
// multi-id SnapshotProductsForWaveDetailed call, replacing the flat
// []ProductDTO returned by SnapshotProductsForWave with enough information
// for the frontend to render "N created, M already existed" feedback.
type SnapshotProductsDetailedResult struct {
	Items        []SnapshotProductDetailItem `json:"items"`
	CreatedCount int                         `json:"createdCount"`
	SkippedCount int                         `json:"skippedCount"`
}
