import type {
  IdentityType,
  PlatformKind,
  InputFactKind,
  TemplateDirection,
  WaveCloseResult,
  EntitlementSelectorType,
  FulfillmentSourceKind,
  BlockReason,
  SupplierOrderStatus,
  DuplicateVerdict,
  WritebackStatus,
  WorkState,
} from '@/shared/api/generated/enums'

export interface Platform {
  ID: number
  Key: string
  Name: string
  Kind: PlatformKind | string
  Notes?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface CustomerProfile {
  ID: number
  DisplayName: string
  Notes?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface PlatformIdentity {
  ID: number
  CustomerProfileID?: number | null
  PlatformID: number
  IdentityType: IdentityType | string
  IdentityValue: string
  NormalizedValue?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface RecipientAddress {
  ID: number
  CustomerProfileID: number
  Label?: string
  RecipientName: string
  Phone?: string
  Country?: string
  Province?: string
  City?: string
  District?: string
  AddressLine1: string
  AddressLine2?: string
  PostalCode?: string
  IsDefault: boolean
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface AddressSnapshot {
  source_address_id?: number | null
  recipient_name: string
  phone?: string
  country?: string
  province?: string
  city?: string
  district?: string
  address_line1: string
  address_line2?: string
  postal_code?: string
}

export interface ProductItem {
  ID: number
  Name: string
  FactoryPlatformID: number
  FactorySKU: string
  Notes?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface ProductAlias {
  ID: number
  ProductItemID: number
  PlatformID: number
  ExternalProductID: string
  Title: string
  Spec?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface ProductBundleComponent {
  ID: number
  AliasID: number
  ProductItemID: number
  Quantity: number
  CreatedAt?: string
  UpdatedAt?: string
}

export interface TemplateConfig {
  ID: number
  PlatformID: number
  DocumentType: string
  Direction: TemplateDirection | string
  Name: string
  Version: number
  Builtin: boolean
  MappingJSON?: string
  LayoutJSON?: string
  Notes?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface CarrierMapping {
  ID: number
  PlatformID: number
  ExternalCode: string
  InternalCode: string
  InternalName: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface InputDocument {
  ID: number
  PlatformID: number
  DocumentType: string
  Direction: TemplateDirection | string
  OriginalName: string
  RawPayload?: string
  TemplateID?: number | null
  TemplateVersion: number
  ImportedAt?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface InputFact {
  ID: number
  DocumentID?: number | null
  PlatformID: number
  Kind: InputFactKind | string
  StableExternalID?: string
  CustomerProfileID?: number | null
  PlatformIdentityID?: number | null
  MembershipLevel?: string
  SourceDocumentNo?: string
  SourceCreatedAt?: string | null
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface InputFactLine {
  ID: number
  FactID: number
  SourceLineNo: number
  ExternalSKU?: string
  ExternalTitle?: string
  ExternalSpec?: string
  ProductItemID?: number | null
  Quantity: number
  WaveID?: number | null
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface DuplicateObservation {
  ID: number
  DocumentID: number
  ExistingFactID: number
  Verdict: DuplicateVerdict | string
  Reason?: string
  Decided: boolean
  CreatedAt?: string
  UpdatedAt?: string
}

export interface Wave {
  ID: number
  WaveNo: string
  Name: string
  Notes?: string
  CloseResult: WaveCloseResult | string
  CloseNote?: string
  ClosedAt?: string | null
  ReopenedAt?: string | null
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface EntitlementSelector {
  type: EntitlementSelectorType | string
  platform_id?: number
  level?: string
  instance_id?: number | null
}

export interface EntitlementRule {
  ID: number
  WaveID: number
  ProductID: number
  Selector: EntitlementSelector
  Quantity: number
  Active: boolean
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface EntitlementException {
  ID: number
  WaveID: number
  ProductID: number
  InstanceID: number
  Quantity: number
  Note?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface EntitlementInstance {
  ID: number
  WaveID: number
  InputFactLineID: number
  CustomerProfileID?: number | null
  PlatformIdentityID?: number | null
  MembershipLevel?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface FulfillmentResult {
  ID: number
  WaveID: number
  SourceKind: FulfillmentSourceKind | string
  EntitlementInstanceID?: number | null
  InputFactLineID?: number | null
  InputFactID?: number | null
  CustomerProfileID?: number | null
  ProductItemID?: number | null
  Quantity: number
  Address: AddressSnapshot
  Frozen: boolean
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface ExecutionQuantityLink {
  ID: number
  FulfillmentResultID: number
  SupplierOrderLineID: number
  Quantity: number
  CreatedAt?: string
}

export interface SupplierOrder {
  ID: number
  WaveID: number
  FactoryPlatformID: number
  Status: SupplierOrderStatus | string
  ExportedAt?: string | null
  VoidedAt?: string | null
  ExportPayload?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface SupplierOrderLine {
  ID: number
  SupplierOrderID: number
  ProductItemID: number
  FactorySKU: string
  Quantity: number
  TrackingID: string
  TrackingRetired: boolean
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface Shipment {
  ID: number
  TrackingID: string
  CarrierCode: string
  CarrierName: string
  TrackingNo: string
  ShippedAt?: string | null
  Quantity: number
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface ChannelWritebackItem {
  ID: number
  InputFactID: number
  ShipmentID: number
  TrackingNo: string
  CarrierCode: string
  Status: WritebackStatus | string
  ErrorMessage?: string
  Payload?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface AppSettings {
  ID?: number
  Locale: string
  Theme: string
  Density: string
  DuplicateRecordMinutes: number
  DuplicateAskDays: number
  UpdatedAt?: string
}

export interface HomeBuckets {
  Unassigned: number
  DuplicateAsk: number
  AlignmentConflict: number
  IdentityUnattached: number
  BlockedResults: number
  WritebackFailed: number
  ResidualClose: number
  RecentWaves: Wave[]
}

export interface InboxRow {
  Line: InputFactLine
  Fact: InputFact
  Document?: InputDocument | null
  Assigned: boolean
  Unaligned: boolean
  Unattached: boolean
}

export interface ResultView {
  Result: FulfillmentResult
  WorkState: WorkState
  Blocks: BlockReason[]
  InFactory: boolean
  Shipped: boolean
  WritebackFailed: boolean
}

export interface ProductTotal {
  ProductID: number
  Name: string
  Quantity: number
}

export interface IngestLine {
  SourceLineNo: number
  ExternalSKU: string
  ExternalTitle: string
  ExternalSpec: string
  Quantity: number
}

export interface IngestFactInput {
  Kind: string
  StableExternalID?: string
  IdentityType: string
  IdentityValue: string
  MembershipLevel?: string
  SourceDocumentNo?: string
  SourceCreatedAt?: string | null
  CustomerProfileID?: number | null
  Lines: IngestLine[]
}

export interface IngestDocumentResult {
  Document: InputDocument
  Duplicates: DuplicateObservation[]
}

export interface GenerateFactoryOrderResult {
  Order: SupplierOrder
  Lines: SupplierOrderLine[]
}
