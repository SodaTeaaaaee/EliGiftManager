package dto

type WaveOverviewDTO struct {
	Wave                    WaveDTO `json:"wave"`
	DemandCount             int     `json:"demandCount"`
	FulfillmentCount        int     `json:"fulfillmentCount"`
	SupplierOrderCount      int     `json:"supplierOrderCount"`
	ShipmentCount           int     `json:"shipmentCount"`
	TrackedFulfillmentCount int     `json:"trackedFulfillmentCount"`

	// Demand-line intake buckets — answers "what should the user do next?"
	AcceptedReadyOrNotRequired int `json:"acceptedReadyOrNotRequired"`
	AcceptedWaitingForInput    int `json:"acceptedWaitingForInput"`
	DeferredCount              int `json:"deferredCount"`
	ExcludedManualCount        int `json:"excludedManualCount"`
	ExcludedDuplicateCount     int `json:"excludedDuplicateCount"`
	ExcludedRevokedCount       int `json:"excludedRevokedCount"`
	MappingBlockedCount        int `json:"mappingBlockedCount"`

	ChannelSyncJobCount            int    `json:"channelSyncJobCount"`
	ChannelSyncPendingCount        int    `json:"channelSyncPendingCount"`
	ChannelSyncRunningCount        int    `json:"channelSyncRunningCount"`
	ChannelSyncSuccessCount        int    `json:"channelSyncSuccessCount"`
	ChannelSyncPartialSuccessCount int    `json:"channelSyncPartialSuccessCount"`
	ChannelSyncFailedCount         int    `json:"channelSyncFailedCount"`
	ManualClosureDecisionCount     int    `json:"manualClosureDecisionCount"`
	ManualUnsupportedCount         int    `json:"manualUnsupportedCount"`
	ManualSkippedCount             int    `json:"manualSkippedCount"`
	ManualCompletedCount           int    `json:"manualCompletedCount"`
	ProjectedLifecycleStage        string `json:"projectedLifecycleStage"`

	BasisDriftSignals      []BasisDriftSignalDTO `json:"basisDriftSignals"`
	HasDriftedBasis        bool                  `json:"hasDriftedBasis"`
	HasRequiredReviewBasis bool                  `json:"hasRequiredReviewBasis"`
}
