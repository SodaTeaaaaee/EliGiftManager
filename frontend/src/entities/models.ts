// Type facade over the Wails v3 generated models.
//
// Source of truth: the generated bindings under frontend/bindings/, imported
// exclusively by src/shared/api/bridge.ts, which re-exports their types with a
// `Wire` suffix (see bridge.ts). This facade narrows those wire shapes into
// the names the UI already uses, so pages keep importing from
// '@/entities/models' and never touch the generated modules (guardrails
// enforces that for all of src/, exempting bridge.ts itself).
//
// Deliberate deviations from the raw generated shapes — each kept because the
// existing UI contract depends on it:
//   - Go nil slices/maps serialize as `null`, so the generator types them
//     `T | null`. The bridge normalizes those to empty containers before they
//     reach the UI, so the facade declares them non-null.
//   - Enum fields keep the string-literal unions from
//     @/shared/api/generated/enums: pages and the i18n glossary compare raw
//     literals (`view.WorkState === 'blocked'`). The generated TS enum objects
//     (BlockReason/WorkState with $zero members) are intentionally NOT
//     re-exported to avoid a duplicate-identifier clash with those unions.
//   - A few timestamp/ID fields stay optional so pages can construct literals
//     (e.g. the settings form, quantity-split components pending backend ids).

import type {
  AppSettingsWire,
  DuplicateObservationWire,
  ExportFileResultWire,
  GenerateFactoryOrderResultWire,
  HomeBucketsWire,
  ImportCarrierMappingsResultWire,
  ImportFileResultWire,
  ImportShipmentFileResultWire,
  IngestDocumentResultWire,
  ParseIssueWire,
  PreviewRowWire,
  QuantitySplitRuleWire,
  ResultViewWire,
  SampleFileInfoWire,
  ShipmentWire,
  SkippedShipmentWire,
  SupplierOrderLineWire,
  TemplatePreviewWire,
  WaveWire,
} from '@/shared/api/bridge'

import type { BlockReason, WorkState } from '@/shared/api/generated/enums'

// ── Domain entities: pure re-exports of the generated wire types ──

export type {
  AddressSnapshotWire as AddressSnapshot,
  CarrierMappingWire as CarrierMapping,
  ChannelWritebackItemWire as ChannelWritebackItem,
  CustomerProfileWire as CustomerProfile,
  DuplicateObservationWire as DuplicateObservation,
  EntitlementExceptionWire as EntitlementException,
  EntitlementRuleWire as EntitlementRule,
  EntitlementSelectorWire as EntitlementSelector,
  FulfillmentResultWire as FulfillmentResult,
  InputDocumentWire as InputDocument,
  InputFactWire as InputFact,
  InputFactLineWire as InputFactLine,
  PlatformWire as Platform,
  ProductAliasWire as ProductAlias,
  ProductBundleComponentWire as ProductBundleComponent,
  ProductItemWire as ProductItem,
  RecipientAddressWire as RecipientAddress,
  ShipmentWire as Shipment,
  SupplierOrderWire as SupplierOrder,
  SupplierOrderLineWire as SupplierOrderLine,
  TemplateConfigWire as TemplateConfig,
  WaveWire as Wave,
} from '@/shared/api/bridge'

// ── App DTOs: pure re-exports of the generated wire types ──

export type {
  DocumentTypeInfoWire as DocumentTypeInfo,
  ExceptionViewWire as ExceptionView,
  IngestFactInputWire as IngestFactInput,
  IngestLineWire as IngestLine,
  InboxRowWire as InboxRow,
  InstanceViewWire as InstanceView,
  ProductTotalWire as ProductTotal,
  ParseIssueWire as ParseIssue,
  SkippedShipmentWire as SkippedShipment,
} from '@/shared/api/bridge'

// ── Frontend-only types (no generated counterpart) ──

/**
 * Platform identity rows are not returned by any bound method, so the v3
 * generator never emits them; kept as a frontend view for identity display.
 */
export interface PlatformIdentity {
  ID: number
  CustomerProfileID?: number | null
  PlatformID: number
  IdentityType: string
  IdentityValue: string
  NormalizedValue?: string
  ExtraData?: string
  CreatedAt?: string
  UpdatedAt?: string
}

// ── Bridge-normalized views over the generated wire types ──
// Go nil-slice `| null` unions are stripped where the bridge rebuilds the
// result; optionality is kept where the UI constructs partial literals.

/** Settings form shape: ID/timestamps are backend-assigned and often absent. */
export interface AppSettings extends Omit<AppSettingsWire, 'ID' | 'UpdatedAt'> {
  ID?: number
  UpdatedAt?: string
}

/** Split components are constructed client-side before the backend assigns ids. */
export interface QuantitySplitComponent {
  ID: number
  RuleID: number
  ProductItemID: number
  Quantity: number
  CreatedAt?: string
  UpdatedAt?: string
}

/** Quantity split rule with non-null components (backend always materializes them). */
export interface QuantitySplitRule extends Omit<QuantitySplitRuleWire, 'Components'> {
  Components: QuantitySplitComponent[]
}

/** Home counters with a non-null recent-wave list (backend sends []). */
export interface HomeBuckets extends Omit<HomeBucketsWire, 'RecentWaves'> {
  RecentWaves: WaveWire[]
}

/**
 * Fulfillment responsibility view. WorkState/Blocks keep the string-literal
 * unions (see module header); the backend always sends a Blocks array.
 */
export interface ResultView extends Omit<ResultViewWire, 'WorkState' | 'Blocks'> {
  WorkState: WorkState
  Blocks: BlockReason[]
}

/** Import receipt with normalized (non-null) duplicate/issue lists. */
export interface ImportFileResult extends Omit<ImportFileResultWire, 'Duplicates' | 'Issues'> {
  Duplicates: DuplicateObservationWire[]
  Issues: ParseIssueWire[]
}

/** Manual-ingest receipt with a normalized duplicate list. */
export interface IngestDocumentResult extends Omit<IngestDocumentResultWire, 'Duplicates'> {
  Duplicates: DuplicateObservationWire[]
}

/** Carrier-table import receipt with a normalized issue list. */
export interface ImportCarrierMappingsResult
  extends Omit<ImportCarrierMappingsResultWire, 'Issues'> {
  Issues: ParseIssueWire[]
}

/** Shipment-return import receipt with normalized lists. */
export interface ImportShipmentFileResult
  extends Omit<ImportShipmentFileResultWire, 'Skipped' | 'Shipments' | 'Issues'> {
  Skipped: SkippedShipmentWire[]
  Shipments: ShipmentWire[]
  Issues: ParseIssueWire[]
}

/** Factory-order submission result with a normalized line list. */
export interface GenerateFactoryOrderResult
  extends Omit<GenerateFactoryOrderResultWire, 'Lines'> {
  Lines: SupplierOrderLineWire[]
}

/** Rendered export receipt with a normalized row list. */
export interface ExportFileResult extends Omit<ExportFileResultWire, 'Rows'> {
  Rows: Record<string, string>[]
}

/** One parsed sample row with non-null semantic values. */
export interface PreviewRow extends Omit<PreviewRowWire, 'Values'> {
  Values: Record<string, string>
}

/** Template test preview with normalized row/issue lists. */
export interface TemplatePreview extends Omit<TemplatePreviewWire, 'Rows' | 'Issues'> {
  Rows: PreviewRow[]
  Issues: ParseIssueWire[]
}

/** Raw sample-file inspection with normalized sheet/record lists. */
export interface SampleFileInfo extends Omit<SampleFileInfoWire, 'Sheets' | 'Records'> {
  Sheets: string[]
  Records: string[][]
}
