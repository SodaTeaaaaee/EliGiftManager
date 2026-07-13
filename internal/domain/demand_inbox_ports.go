package domain

import "context"

// DemandInboxAssignmentRepository provides bulk wave-demand-assignment lookups keyed by a
// batch of demand document IDs. It exists to let the demand inbox query (ListDemandInboxRows)
// fetch assignment state for many documents in a single round trip instead of issuing one
// ListByDemandDocument call per document (the former N+1 pattern).
type DemandInboxAssignmentRepository interface {
	ListByDemandDocumentIDs(ctx context.Context, docIDs []uint) ([]WaveDemandAssignment, error)
}

// DemandInboxLineRepository provides bulk demand-line lookups keyed by a batch of demand
// document IDs. It exists to let the demand inbox query fetch line-level rollup stats for
// many documents in a single round trip instead of one ListLinesByDocument call per document.
type DemandInboxLineRepository interface {
	ListLinesByDocumentIDs(ctx context.Context, docIDs []uint) ([]DemandLine, error)
}
