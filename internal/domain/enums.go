package domain

type IdentityType string

const (
	IdentityTypePlatformUID     IdentityType = "platform_uid"
	IdentityTypeEmail           IdentityType = "email"
	IdentityTypeUsername        IdentityType = "username"
	IdentityTypeExternalBuyerID IdentityType = "external_buyer_id"
)

type PlatformKind string

const (
	PlatformKindSource  PlatformKind = "source"
	PlatformKindFactory PlatformKind = "factory"
)

type InputFactKind string

const (
	InputFactKindMembership    InputFactKind = "membership"
	InputFactKindRetailOrder   InputFactKind = "retail_order"
	InputFactKindOperatorGrant InputFactKind = "operator_grant"
)

type TemplateDirection string

const (
	TemplateDirectionInput  TemplateDirection = "input"
	TemplateDirectionOutput TemplateDirection = "output"
)

type WaveCloseResult string

const (
	WaveCloseResultOpen     WaveCloseResult = "open"
	WaveCloseResultClean    WaveCloseResult = "clean"
	WaveCloseResultResidual WaveCloseResult = "residual"
)

type EntitlementSelectorType string

const (
	SelectorPlatformLevel EntitlementSelectorType = "platform_level"
	SelectorWaveAll       EntitlementSelectorType = "wave_all"
	SelectorInstance      EntitlementSelectorType = "instance"
)

type FulfillmentSourceKind string

const (
	SourceEntitlementInstance FulfillmentSourceKind = "entitlement_instance"
	SourceRetailLine          FulfillmentSourceKind = "retail_line"
	SourceOperatorGrant       FulfillmentSourceKind = "operator_grant"
)

type BlockReason string

const (
	BlockUnalignedProduct        BlockReason = "unaligned_product"
	BlockUnusableAddress         BlockReason = "unusable_address"
	BlockIdentityUnattached      BlockReason = "identity_unattached"
	BlockQuantitySplitNotSumming BlockReason = "quantity_split_not_summing"
)

type SupplierOrderStatus string

const (
	SupplierOrderGenerated SupplierOrderStatus = "generated"
	SupplierOrderExported  SupplierOrderStatus = "exported"
	SupplierOrderVoided    SupplierOrderStatus = "voided"
)

type DuplicateVerdict string

const (
	DuplicateRecordOnly        DuplicateVerdict = "record_only"
	DuplicateAskOperator       DuplicateVerdict = "ask_operator"
	DuplicateNewResponsibility DuplicateVerdict = "new_responsibility"
)

type WritebackStatus string

const (
	WritebackPending WritebackStatus = "pending"
	WritebackSent    WritebackStatus = "sent"
	WritebackFailed  WritebackStatus = "failed"
)

type WorkState string

const (
	WorkStateBlocked         WorkState = "blocked"
	WorkStateReady           WorkState = "ready"
	WorkStateInFactory       WorkState = "in_factory"
	WorkStateShipped         WorkState = "shipped"
	WorkStateWritebackFailed WorkState = "writeback_failed"
)
