/**
 * FilterBar kit family — shared type contract (plan section 3.3.2 / 4.3).
 *
 * A "filter schema" is a small declarative array describing the filterable
 * dimensions of a page (e.g. the fulfillment grid's four status dimensions +
 * a keyword search). `useUrlFilters` turns a schema into URL-synced reactive
 * state; `FilterBar.vue` renders controls from the same schema;
 * `SavedViews.vue` persists/restores named snapshots of that state.
 *
 * IMPORTANT: declare schemas with `as const` so field `key`s stay literal
 * string types and `FilterState<S>` / `FilterKey<S>` narrow correctly:
 *
 * ```ts
 * const schema = [
 *   { key: 'addressState', type: 'enum-multi', dimension: 'addressState' },
 *   { key: 'keyword', type: 'keyword' },
 * ] as const satisfies FilterSchema
 * ```
 */
import type { ComputedRef } from 'vue'
import type { GlossaryDimension } from '@/shared/i18n/glossary'

/** A multi-select filter over one of the glossary's status dimensions. */
export interface EnumMultiFilterField {
  key: string
  type: 'enum-multi'
  /** Which glossary dimension supplies the option labels/tones. */
  dimension: GlossaryDimension
  /** Restrict the offered options to a subset of the dimension's values (default: all). */
  values?: readonly string[]
}

/** A free-text keyword filter. Debouncing is FilterBar.vue's responsibility. */
export interface KeywordFilterField {
  key: string
  type: 'keyword'
}

export type FilterField = EnumMultiFilterField | KeywordFilterField

/** A schema is just an ordered list of fields. Declare with `as const`. */
export type FilterSchema = readonly FilterField[]

/** Union of every field key declared in a schema. */
export type FilterKey<S extends FilterSchema> = S[number]['key']

/** Per-field value type: `string[]` for enum-multi, `string` for keyword. */
type FieldValue<F> = F extends { type: 'enum-multi' } ? string[] : string

/**
 * The reactive state shape a schema produces: one property per field key,
 * typed per that field's kind. Requires the schema to be declared with
 * `as const` (or `satisfies FilterSchema` on a `const` literal) so the
 * mapped type can key off literal string keys instead of the generic `string`.
 */
export type FilterState<S extends FilterSchema> = {
  [F in S[number] as F['key']]: FieldValue<F>
}

/**
 * A loose, JSON-serializable snapshot of a filter state — what gets written
 * to a saved view / preset / the URL round-trip. Deliberately not generic
 * over a schema so it can be stored/passed around without type ceremony
 * (SavedViews.vue does not need to know the originating schema).
 */
export type FilterSnapshot = Record<string, string[] | string>

/** A single removable chip describing one active filter value. */
export interface ActiveFilterChip {
  /** Stable identity for `v-for` — `${fieldKey}:${value}` for enum-multi, `fieldKey` for keyword. */
  id: string
  fieldKey: string
  type: FilterField['type']
  /** The glossary dimension, when this chip represents an enum-multi value. */
  dimension?: GlossaryDimension
  /** The raw enum value (enum-multi chips) or the keyword text (keyword chips). */
  value: string
}

/** A named, non-deletable, pre-built filter view (plan 3.3.2's 阻塞项/可提交 pattern). */
export interface FilterViewPreset {
  id: string
  /** Display label — already resolved (i18n'd) by the caller. */
  label: string
  snapshot: FilterSnapshot
}

/** A user-saved, deletable filter view persisted to localStorage. */
export interface SavedFilterView {
  id: string
  name: string
  snapshot: FilterSnapshot
  createdAt: number
}

/**
 * The plain, non-generic surface `FilterBar.vue` and `SavedViews.vue`
 * actually consume. `useUrlFilters<S>()`'s return type (`UseUrlFiltersApi<S>`)
 * is structurally assignable here without a cast: the mutator methods are
 * declared with method-shorthand syntax (bivariant parameter checking, so a
 * method narrowed to a literal `FilterKey<S>` union satisfies a plain
 * `string` parameter), `schema: S` widens trivially to the `FilterSchema`
 * base type, and `state: FilterState<S>` (a literal-keyed mapped type)
 * satisfies an index-signature target because every value type in it is
 * still assignable to `unknown`.
 *
 * Components are deliberately NOT generic Vue SFCs over `S` — the schema's
 * literal key safety is only useful at the call site that builds the
 * `useUrlFilters(schema)` object; once handed to a renderer, every field is
 * driven by runtime `field.key` strings anyway, so a generic component would
 * add real fragility (Vue generic-SFC + slots type inference is still a
 * rough edge) for no practical safety gain.
 */
export interface FilterBarFilters {
  schema: FilterSchema
  state: Record<string, unknown>
  setEnumValues(key: string, values: readonly string[]): void
  toggleEnumValue(key: string, value: string): void
  setKeyword(key: string, value: string): void
  clearField(key: string): void
  clearAll(): void
  activeCount: ComputedRef<number>
  isActive: ComputedRef<boolean>
  activeChips: ComputedRef<ActiveFilterChip[]>
  getSnapshot(): FilterSnapshot
  applySnapshot(snapshot: FilterSnapshot): void
}
