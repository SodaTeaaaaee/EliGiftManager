/**
 * WizardFrame kit family — shared type contract (P4 demand-intake wizard,
 * and future multi-step flows e.g. a P5 shipment-CSV mapping wizard).
 *
 * WizardFrame is presentational + controlled: the parent page owns the
 * `current` step key and all step-body state; WizardFrame only renders the
 * step indicator + the current step's body (via the default slot) + a nav
 * bar, and emits intent events (`next` / `back` / `finish` / `cancel`).
 * WizardFrame never mutates `current` itself except via the optional
 * `update:current` emit when the operator clicks an already-completed
 * step's indicator to jump back.
 */

/** One step in the wizard's linear sequence. */
export interface WizardStep {
  /** Stable identifier, matched against `current` to determine the active step. */
  key: string
  /** Already-resolved display title (call `t(...)` before building the list — mirrors DataGrid's `title` convention). */
  title: string
}
