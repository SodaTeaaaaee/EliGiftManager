package dto

type DemandDocumentDTO struct {
	ID                   uint   `json:"id"`
	Kind                 string `json:"kind"`
	CaptureMode          string `json:"captureMode"`
	SourceChannel        string `json:"sourceChannel"`
	SourceSurface        string `json:"sourceSurface"`
	IntegrationProfileID *uint  `json:"integrationProfileId"`
	SourceDocumentNo     string `json:"sourceDocumentNo"`
	SourceCustomerRef    string `json:"sourceCustomerRef"`
	CustomerProfileID    *uint  `json:"customerProfileId"`
	SourceCreatedAt      string `json:"sourceCreatedAt"`
	SourcePaidAt         string `json:"sourcePaidAt"`
	Currency             string `json:"currency"`
	AuthoritySnapshotAt  string `json:"authoritySnapshotAt"`
	RawPayload           string `json:"rawPayload"`
	ExtraData            string `json:"extraData"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type DemandLineDTO struct {
	ID                    uint   `json:"id"`
	DemandDocumentID      uint   `json:"demandDocumentId"`
	SourceLineNo          *int   `json:"sourceLineNo"`
	LineType              string `json:"lineType"`
	ObligationTriggerKind string `json:"obligationTriggerKind"`
	EntitlementAuthority  string `json:"entitlementAuthority"`
	RecipientInputState   string `json:"recipientInputState"`
	RoutingDisposition    string `json:"routingDisposition"`
	RoutingReasonCode     string `json:"routingReasonCode"`
	EligibilityContextRef  string `json:"eligibilityContextRef"`
	ProductMasterID       *uint  `json:"productMasterId"`
	ExternalTitle         string `json:"externalTitle"`
	RequestedQuantity     int    `json:"requestedQuantity"`
	EntitlementCode       string `json:"entitlementCode"`
	GiftLevelSnapshot     string `json:"giftLevelSnapshot"`
	RecipientInputPayload string `json:"recipientInputPayload"`
	RawPayload            string `json:"rawPayload"`
	ExtraData             string `json:"extraData"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
}

type CreateDemandInput struct {
	Kind                 string                 `json:"kind"`
	CaptureMode          string                 `json:"captureMode"`
	SourceChannel        string                 `json:"sourceChannel"`
	SourceSurface        string                 `json:"sourceSurface"`
	SourceDocumentNo     string                 `json:"sourceDocumentNo"`
	SourceCustomerRef    string                 `json:"sourceCustomerRef"`
	CustomerProfileID    *uint                  `json:"customerProfileId"`
	IntegrationProfileID *uint                  `json:"integrationProfileId"`
	Lines                []CreateDemandLineInput `json:"lines"`
}

type CreateDemandLineInput struct {
	LineType              string `json:"lineType"`
	ObligationTriggerKind string `json:"obligationTriggerKind"`
	EntitlementAuthority  string `json:"entitlementAuthority"`
	RecipientInputState   string `json:"recipientInputState"`
	RoutingDisposition    string `json:"routingDisposition"`
	RoutingReasonCode     string `json:"routingReasonCode"`
	EligibilityContextRef  string `json:"eligibilityContextRef"`
	EntitlementCode       string `json:"entitlementCode"`
	GiftLevelSnapshot     string `json:"giftLevelSnapshot"`
	ProductMasterID       *uint  `json:"productMasterId"`
	RecipientInputPayload string `json:"recipientInputPayload"`
	ExternalTitle         string `json:"externalTitle"`
	RequestedQuantity     int    `json:"requestedQuantity"`
}

// DemandMappingBlockedLine records a demand line that could not be mapped
// to a fulfillment line because of a missing or unresolvable product reference.
type DemandMappingBlockedLine struct {
	DemandLineID  uint   `json:"demandLineId"`
	DemandLineTitle string `json:"demandLineTitle"`
	Reason        string `json:"reason"`
}

// DemandMappingResult contains the outcome of a demand-driven mapping run.
type DemandMappingResult struct {
	CreatedLines []FulfillmentLineDTO        `json:"createdLines"`
	BlockedLines []DemandMappingBlockedLine  `json:"blockedLines"`
}

type DemandInboxFilterInput struct {
	Assignment string `json:"assignment"`
	DemandKind string `json:"demandKind"`
}
