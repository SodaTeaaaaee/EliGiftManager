package model

// BatchValidationMissingMember 表示批次预校验时缺少有效地址的会员。
type BatchValidationMissingMember struct {
	MemberID       uint   `json:"memberId"`
	Platform       string `json:"platform"`
	PlatformUID    string `json:"platformUid"`
	LatestNickname string `json:"latestNickname"`
}

// BatchValidationResult 表示批次导出前的地址预校验结果。
type BatchValidationResult struct {
	BatchName             string                         `json:"batchName"`
	TotalRecords          int                            `json:"totalRecords"`
	BoundAddressRecords   int                            `json:"boundAddressRecords"`
	PendingAddressRecords int                            `json:"pendingAddressRecords"`
	MissingMembers        []BatchValidationMissingMember `json:"missingMembers"`
}
