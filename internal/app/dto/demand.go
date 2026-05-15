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
	Kind              string                 `json:"kind"`
	CaptureMode       string                 `json:"captureMode"`
	SourceChannel     string                 `json:"sourceChannel"`
	SourceDocumentNo  string                 `json:"sourceDocumentNo"`
	SourceCustomerRef string                 `json:"sourceCustomerRef"`
	CustomerProfileID *uint                  `json:"customerProfileId"`
	Lines             []CreateDemandLineInput `json:"lines"`
}

type CreateDemandLineInput struct {
	LineType              string `json:"lineType"`
	ObligationTriggerKind string `json:"obligationTriggerKind"`
	EntitlementAuthority  string `json:"entitlementAuthority"`
	RoutingDisposition    string `json:"routingDisposition"`
	ExternalTitle         string `json:"externalTitle"`
	RequestedQuantity     int    `json:"requestedQuantity"`
}
