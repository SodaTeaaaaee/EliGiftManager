package domain

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

// FulfillmentLine: AllocationState
type AllocationState string

const (
	AllocationStateDraft AllocationState = "draft"
	AllocationStateReady AllocationState = "ready"
)

// FulfillmentLine: AddressState
type AddressState string

const (
	AddressStateMissing AddressState = "missing"
	AddressStateReady   AddressState = "ready"
	AddressStateInvalid AddressState = "invalid"
)

// FulfillmentLine: SupplierState
type SupplierState string

const (
	SupplierStateNotSubmitted     SupplierState = "not_submitted"
	SupplierStateSubmitted        SupplierState = "submitted"
	SupplierStateAccepted         SupplierState = "accepted"
	SupplierStateProducing        SupplierState = "producing"
	SupplierStatePartiallyShipped SupplierState = "partially_shipped"
	SupplierStateShipped          SupplierState = "shipped"
	SupplierStateCanceled         SupplierState = "canceled"
)

// FulfillmentLine: ChannelSyncState
type ChannelSyncState string

const (
	ChannelSyncStateNotRequired     ChannelSyncState = "not_required"
	ChannelSyncStateUnsupported     ChannelSyncState = "unsupported"
	ChannelSyncStatePending         ChannelSyncState = "pending"
	ChannelSyncStateSynced          ChannelSyncState = "synced"
	ChannelSyncStateManualConfirmed ChannelSyncState = "manual_confirmed"
	ChannelSyncStateSkipped         ChannelSyncState = "skipped"
	ChannelSyncStateFailed          ChannelSyncState = "failed"
)

// Wave: LifecycleStage
type LifecycleStage string

const (
	LifecycleStageIntake          LifecycleStage = "intake"
	LifecycleStageAllocation      LifecycleStage = "allocation"
	LifecycleStageReview          LifecycleStage = "review"
	LifecycleStageExecution       LifecycleStage = "execution"
	LifecycleStageSyncingBack     LifecycleStage = "syncing_back"
	LifecycleStageAwaitingClosure LifecycleStage = "awaiting_manual_closure"
	LifecycleStageClosed          LifecycleStage = "closed"
)

// History command kinds — user-intent operations only
const (
	CmdSystemBaseline       = "_system_baseline"
	CmdAssignDemand         = "assign_demand"
	CmdGenerateParticipants = "generate_participants"
	CmdMapDemandLines       = "map_demand_lines"
	CmdReconcileWave        = "reconcile_wave"
	CmdCreateRule           = "create_rule"
	CmdUpdateRule           = "update_rule"
	CmdDeleteRule           = "delete_rule"
	CmdRecordAdjustment     = "record_adjustment"
)

// ProductMaster: ProductKind
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
