/**
 * Operator roster — shared, localStorage-persisted list of operator ids
 * (most-recent-first), consumed by adjustment/closure forms as a
 * replacement for free-text `operatorId` entry (plan:261 "操作员名单").
 *
 * Reuses the EXACT localStorage key already written by P3's
 * `BatchAdjustDialog.vue` / `RowDetailDrawer.vue`
 * (`eligiftmanager:recent-operator-ids`) — do NOT change this key, or
 * already-recorded operator history is lost on upgrade. Those two
 * components, plus P5's `ManualClosureForm.vue`, are retrofitted onto this
 * module by the settings unit; the settings page itself is the CRUD surface
 * (add/remove) backed by this same store.
 *
 * Backend contract is unchanged — every `*Input` DTO still carries
 * `operatorId: string` as plain free text server-side. This store only
 * upgrades the FRONTEND input widget from ad-hoc-per-component localStorage
 * reads to one shared, reactive list.
 */
import { defineStore } from "pinia";
import { ref } from "vue";

const STORAGE_KEY = "eligiftmanager:recent-operator-ids";
// Generous cap for a full CRUD roster (vs. the original 8-entry "recently
// used" cap the P3 duplicates used) — most-recent-first, de-duplicated.
const MAX_ROSTER_SIZE = 50;

function readStorage(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((entry): entry is string => typeof entry === "string");
  } catch {
    // localStorage can throw in locked-down / privacy-mode contexts, and
    // JSON.parse can throw on corrupt data — fall back to an empty roster
    // rather than crash the app shell.
    return [];
  }
}

function writeStorage(ids: string[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  } catch {
    // best-effort persistence only
  }
}

/**
 * Pinia setup store — `ids` is the reactive, most-recent-first roster for
 * direct template consumption (e.g. a settings-page CRUD list); `list()`/
 * `add()`/`remove()` are the stable functional API documented for
 * consumers that just need to read/mutate without binding to reactivity
 * (e.g. an `NAutoComplete`'s options callback).
 */
export const useOperatorRosterStore = defineStore("operator-roster", () => {
  const ids = ref<string[]>(readStorage());

  /** Current roster, most-recent-first. */
  function list(): string[] {
    return ids.value;
  }

  /**
   * Add (or bump to the front of) an operator id. Trims whitespace,
   * de-duplicates case-sensitively, no-ops on an empty/whitespace-only
   * value. Caps the roster at `MAX_ROSTER_SIZE` entries.
   */
  function add(value: string): void {
    const trimmed = value.trim();
    if (!trimmed) return;
    const rest = ids.value.filter((existing) => existing !== trimmed);
    ids.value = [trimmed, ...rest].slice(0, MAX_ROSTER_SIZE);
    writeStorage(ids.value);
  }

  /** Remove an operator id from the roster (settings-page delete action). */
  function remove(value: string): void {
    ids.value = ids.value.filter((existing) => existing !== value);
    writeStorage(ids.value);
  }

  return { ids, list, add, remove };
});
