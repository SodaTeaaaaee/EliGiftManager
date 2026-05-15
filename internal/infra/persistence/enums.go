package persistence

// CustomerProfile: ProfileType
type ProfileType string

const (
	ProfileTypeMember ProfileType = "member"
	ProfileTypeBuyer  ProfileType = "buyer"
	ProfileTypeMixed  ProfileType = "mixed"
	ProfileTypeManual ProfileType = "manual"
)

// CustomerIdentity: IdentityType
type IdentityType string

const (
	IdentityTypePlatformUID     IdentityType = "platform_uid"
	IdentityTypeEmail           IdentityType = "email"
	IdentityTypeUsername        IdentityType = "username"
	IdentityTypeExternalBuyerID IdentityType = "external_buyer_id"
)

// DemandDocument: Kind
type DemandKind string

const (
	DemandKindMembershipEntitlement DemandKind = "membership_entitlement"
	DemandKindRetailOrder           DemandKind = "retail_order"
)

// DemandDocument: CaptureMode
type CaptureMode string

const (
	CaptureModeDocumentImport CaptureMode = "document_import"
	CaptureModeAPIIngest      CaptureMode = "api_ingest"
	CaptureModeManualEntry    CaptureMode = "manual_entry"
)

// DemandLine: LineType
type DemandLineType string

const (
	DemandLineTypeEntitlementRule DemandLineType = "entitlement_rule"
	DemandLineTypeSKUOrder        DemandLineType = "sku_order"
	DemandLineTypeManualEntry     DemandLineType = "manual_entry"
)

// DemandLine: ObligationTriggerKind
type ObligationTriggerKind string

const (
	ObligationTriggerKindPeriodicMembership         ObligationTriggerKind = "periodic_membership"
	ObligationTriggerKindLoyaltyMembership          ObligationTriggerKind = "loyalty_membership"
	ObligationTriggerKindSupporterOnlyPurchase      ObligationTriggerKind = "supporter_only_purchase"
	ObligationTriggerKindMemberOnlyDiscountPurchase ObligationTriggerKind = "member_only_discount_purchase"
	ObligationTriggerKindCampaignReward             ObligationTriggerKind = "campaign_reward"
	ObligationTriggerKindManualCompensation         ObligationTriggerKind = "manual_compensation"
)

// DemandLine: EntitlementAuthority
type EntitlementAuthority string

const (
	EntitlementAuthorityLocalPolicy      EntitlementAuthority = "local_policy"
	EntitlementAuthorityUpstreamPlatform EntitlementAuthority = "upstream_platform"
	EntitlementAuthorityManualGrant      EntitlementAuthority = "manual_grant"
)

// DemandLine: RecipientInputState
type RecipientInputState string

const (
	RecipientInputStateNotRequired        RecipientInputState = "not_required"
	RecipientInputStateWaitingForInput    RecipientInputState = "waiting_for_input"
	RecipientInputStatePartiallyCollected RecipientInputState = "partially_collected"
	RecipientInputStateReady              RecipientInputState = "ready"
	RecipientInputStateWaived             RecipientInputState = "waived"
	RecipientInputStateExpired            RecipientInputState = "expired"
)

// DemandLine: RoutingDisposition
type RoutingDisposition string

const (
	RoutingDispositionPendingIntake     RoutingDisposition = "pending_intake"
	RoutingDispositionAccepted          RoutingDisposition = "accepted"
	RoutingDispositionDeferred          RoutingDisposition = "deferred"
	RoutingDispositionExcludedManual    RoutingDisposition = "excluded_manual"
	RoutingDispositionExcludedDuplicate RoutingDisposition = "excluded_duplicate"
	RoutingDispositionExcludedRevoked   RoutingDisposition = "excluded_revoked"
)

// Wave: WaveType
type WaveType string

const (
	WaveTypeMembership WaveType = "membership"
	WaveTypeRetail     WaveType = "retail"
	WaveTypeMixed      WaveType = "mixed"
)

// WaveParticipantSnapshot: SnapshotType
type SnapshotType string

const (
	SnapshotTypeMember SnapshotType = "member"
	SnapshotTypeBuyer  SnapshotType = "buyer"
	SnapshotTypeMixed  SnapshotType = "mixed"
)

// FulfillmentLine: LineReason
type FulfillmentLineReason string

const (
	LineReasonEntitlement    FulfillmentLineReason = "entitlement"
	LineReasonRetailOrder    FulfillmentLineReason = "retail_order"
	LineReasonWaveAdjustment FulfillmentLineReason = "wave_adjustment"
)

// SupplierOrder: SubmissionMode
type SubmissionMode string

const (
	SubmissionModeCSV    SubmissionMode = "csv"
	SubmissionModeManual SubmissionMode = "manual"
	SubmissionModeAPI    SubmissionMode = "api"
)

// SupplierOrder: Status
type SupplierOrderStatus string

const (
	SupplierOrderStatusDraft            SupplierOrderStatus = "draft"
	SupplierOrderStatusSubmitted        SupplierOrderStatus = "submitted"
	SupplierOrderStatusAccepted         SupplierOrderStatus = "accepted"
	SupplierOrderStatusPartiallyShipped SupplierOrderStatus = "partially_shipped"
	SupplierOrderStatusShipped          SupplierOrderStatus = "shipped"
	SupplierOrderStatusCanceled         SupplierOrderStatus = "canceled"
)

// Shipment: ShipmentStatus
type ShipmentStatus string

const (
	ShipmentStatusPending   ShipmentStatus = "pending"
	ShipmentStatusShipped   ShipmentStatus = "shipped"
	ShipmentStatusInTransit ShipmentStatus = "in_transit"
	ShipmentStatusDelivered ShipmentStatus = "delivered"
	ShipmentStatusException ShipmentStatus = "exception"
	ShipmentStatusReturned  ShipmentStatus = "returned"
)

// ChannelSyncJob: Direction
type ChannelSyncDirection string

const (
	ChannelSyncDirectionPushTracking ChannelSyncDirection = "push_tracking"
)

// ChannelSyncJob: Status
type ChannelSyncJobStatus string

const (
	ChannelSyncJobStatusPending       ChannelSyncJobStatus = "pending"
	ChannelSyncJobStatusRunning       ChannelSyncJobStatus = "running"
	ChannelSyncJobStatusSuccess       ChannelSyncJobStatus = "success"
	ChannelSyncJobStatusPartialSuccess ChannelSyncJobStatus = "partial_success"
	ChannelSyncJobStatusFailed        ChannelSyncJobStatus = "failed"
)

// ChannelSyncItem: Status
type ChannelSyncItemStatus string

const (
	ChannelSyncItemStatusPending ChannelSyncItemStatus = "pending"
	ChannelSyncItemStatusSuccess ChannelSyncItemStatus = "success"
	ChannelSyncItemStatusFailed  ChannelSyncItemStatus = "failed"
)

// IntegrationProfile: DemandKind (reuses same values as DemandDocument.Kind)
type ProfileDemandKind string

const (
	ProfileDemandKindMembershipEntitlement ProfileDemandKind = "membership_entitlement"
	ProfileDemandKindRetailOrder           ProfileDemandKind = "retail_order"
)

// IntegrationProfile: InitialAllocationStrategy
type InitialAllocationStrategy string

const (
	InitialAllocationStrategyPolicyDriven InitialAllocationStrategy = "policy_driven"
	InitialAllocationStrategyDemandDriven InitialAllocationStrategy = "demand_driven"
)

// IntegrationProfile: IdentityStrategy
type IdentityStrategy string

const (
	IdentityStrategyPlatformUID     IdentityStrategy = "platform_uid"
	IdentityStrategyEmail           IdentityStrategy = "email"
	IdentityStrategyExternalBuyerID IdentityStrategy = "external_buyer_id"
)

// IntegrationProfile: EntitlementAuthorityMode
type EntitlementAuthorityMode string

const (
	EntitlementAuthorityModeLocalPolicy      EntitlementAuthorityMode = "local_policy"
	EntitlementAuthorityModeUpstreamPlatform EntitlementAuthorityMode = "upstream_platform"
	EntitlementAuthorityModeManualGrantOnly  EntitlementAuthorityMode = "manual_grant_only"
)

// IntegrationProfile: RecipientInputMode
type RecipientInputMode string

const (
	RecipientInputModeNone             RecipientInputMode = "none"
	RecipientInputModePlatformClaim    RecipientInputMode = "platform_claim"
	RecipientInputModeExternalForm     RecipientInputMode = "external_form"
	RecipientInputModeManualCollection RecipientInputMode = "manual_collection"
)

// IntegrationProfile: ReferenceStrategy
type ReferenceStrategy string

const (
	ReferenceStrategyMemberLevel    ReferenceStrategy = "member_level"
	ReferenceStrategyOrderLevel     ReferenceStrategy = "order_level"
	ReferenceStrategyOrderLineLevel ReferenceStrategy = "order_line_level"
)

// IntegrationProfile: TrackingSyncMode
type TrackingSyncMode string

const (
	TrackingSyncModeAPIPush            TrackingSyncMode = "api_push"
	TrackingSyncModeDocumentExport     TrackingSyncMode = "document_export"
	TrackingSyncModeManualConfirmation TrackingSyncMode = "manual_confirmation"
	TrackingSyncModeUnsupported        TrackingSyncMode = "unsupported"
)

// IntegrationProfile: ClosurePolicy
type ClosurePolicy string

const (
	ClosurePolicyCloseAfterSync               ClosurePolicy = "close_after_sync"
	ClosurePolicyCloseAfterManualConfirmation ClosurePolicy = "close_after_manual_confirmation"
	ClosurePolicyCloseAfterShipment           ClosurePolicy = "close_after_shipment"
)

// ChannelClosureDecisionRecord: DecisionKind
type ChannelClosureDecisionKind string

const (
	DecisionKindUnsupported       ChannelClosureDecisionKind = "mark_sync_unsupported"
	DecisionKindSkipped           ChannelClosureDecisionKind = "mark_sync_skipped"
	DecisionKindCompletedManually ChannelClosureDecisionKind = "mark_sync_completed_manually"
)

// ---- AdjustmentKind ----

const (
	AdjustmentKindAddSend    = "add_send"
	AdjustmentKindReduceSend = "reduce_send"
	AdjustmentKindReplace    = "replace"
	AdjustmentKindRemove     = "remove"
	AdjustmentKindSupplement = "supplement"
)

// ---- DocumentType ----

const (
	DocumentTypeImportEntitlement    = "import_entitlement"
	DocumentTypeImportSalesOrder     = "import_sales_order"
	DocumentTypeImportProductCatalog = "import_product_catalog"
	DocumentTypeExportSupplierOrder  = "export_supplier_order"
	DocumentTypeImportShipment       = "import_supplier_shipment"
	DocumentTypeExportTracking       = "export_source_tracking_update"
)

// ---- DocumentFormat ----

const (
	DocumentFormatCSV        = "csv"
	DocumentFormatXLSX       = "xlsx"
	DocumentFormatJSON       = "json"
	DocumentFormatAPIPayload = "api_payload"
)

// ---- ProductKind ----

type ProductKind string

const (
	ProductKindBadge    ProductKind = "badge"
	ProductKindStandee  ProductKind = "standee"
	ProductKindCharm    ProductKind = "charm"
	ProductKindPostcard ProductKind = "postcard"
	ProductKindPrint    ProductKind = "print"
	ProductKindBundle   ProductKind = "bundle"
	ProductKindOther    ProductKind = "other"
)
