package dto

// ActionCenterBucketFilterDTO carries the pre-filter selection a deep-link should
// apply when navigating from an action-center card into the wave workspace for a
// given blocked bucket. The four state fields mirror the fulfillment-line grid
// dimensions described in plan 5.4 (AllocationState / AddressState / SupplierState /
// ChannelSyncState) plus ReviewRequirement / Drift, so the frontend can pass this
// straight through once the filtered ListWaveFulfillmentRows overload lands. StepKey
// names the workspace step to land on when the bucket is not a grid-row filter (e.g.
// "waiting_input" surfaces on the Intake step, not the fulfillment grid).
type ActionCenterBucketFilterDTO struct {
	StepKey           string `json:"stepKey,omitempty"`
	AllocationState   string `json:"allocationState,omitempty"`
	AddressState      string `json:"addressState,omitempty"`
	SupplierState     string `json:"supplierState,omitempty"`
	ChannelSyncState  string `json:"channelSyncState,omitempty"`
	ReviewRequirement string `json:"reviewRequirement,omitempty"`
	Drift             string `json:"drift,omitempty"`
}

// ActionCenterWaveBucketDTO is one blocked bucket for a single wave: a bucket kind,
// its count, and the deep-link filter selection a click should apply.
//
// BucketKind is one of: "missing_address" | "waiting_input" | "mapping_blocked" |
// "channel_sync_failed" | "awaiting_manual_closure" | "drift_needs_review".
type ActionCenterWaveBucketDTO struct {
	WaveID     uint                        `json:"waveId"`
	BucketKind string                      `json:"bucketKind"`
	Count      int                         `json:"count"`
	Filter     ActionCenterBucketFilterDTO `json:"filter"`
}

// ActionCenterWaveSummaryDTO groups a wave's non-zero blocked buckets. Waves with no
// blocked buckets (nothing actionable) are omitted from ActionCenterSummaryDTO.Waves
// entirely, and closed waves are never included.
type ActionCenterWaveSummaryDTO struct {
	WaveID            uint                        `json:"waveId"`
	WaveNo            string                      `json:"waveNo"`
	WaveName          string                      `json:"waveName"`
	Buckets           []ActionCenterWaveBucketDTO `json:"buckets"`
	TotalBlockedCount int                         `json:"totalBlockedCount"`
}

// ActionCenterNavBadgeDTO is the live workload badge for one top-level nav item.
// NavKey is one of: "home" | "waves" | "inbox" | "customers" | "products" |
// "integrations". Items without a defined backend signal yet (customers / products /
// integrations) report Count 0 rather than a fabricated value.
type ActionCenterNavBadgeDTO struct {
	NavKey string `json:"navKey"`
	Count  int    `json:"count"`
}

// ActionCenterSummaryDTO is the single aggregated payload behind the Action Center
// (Home) page: per-wave blocked-bucket counts with deep-link filters, the inbox
// pending-intake count, and per-nav-item badge counts.
type ActionCenterSummaryDTO struct {
	Waves                   []ActionCenterWaveSummaryDTO `json:"waves"`
	InboxPendingIntakeCount int                           `json:"inboxPendingIntakeCount"`
	NavBadges               []ActionCenterNavBadgeDTO    `json:"navBadges"`
}
