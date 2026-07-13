package dto

type UndoCustomerMergeInput struct {
	MergeID uint `json:"mergeId"`
}

type UndoCustomerMergeResult struct {
	MergeID                     uint `json:"mergeId"`
	RestoredSourceProfileID     uint `json:"restoredSourceProfileId"`
	TargetProfileID             uint `json:"targetProfileId"`
	RestoredIdentityCount       int  `json:"restoredIdentityCount"`
	RestoredAddressCount        int  `json:"restoredAddressCount"`
	RestoredDemandDocumentCount int  `json:"restoredDemandDocumentCount"`
}
