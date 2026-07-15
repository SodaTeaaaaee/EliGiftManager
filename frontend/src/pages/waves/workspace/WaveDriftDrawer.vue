<script setup lang="ts">
/**
 * WaveDriftDrawer — the wave-workspace overview's basis-drift detail panel
 * (plan 3.3.1, P2 unit B). Lists every `BasisDriftSignalDTO` on the
 * injected snapshot's `overview.basisDriftSignals`: which basis (supplier
 * order / shipment / channel sync) drifted, how urgently it needs review,
 * and why.
 *
 * Per-signal severity is derived LOCALLY into a `driftSummary`-compatible
 * value (P2 foundations contract decision #3) instead of adding a new
 * glossary dimension — `basisDriftStatus` only carries two raw values
 * (`"in_sync"` / `"drifted"`), so `reviewRequirement` supplies the severity
 * gradient that `<StatusBadge dimension="driftSummary">` expects.
 *
 * Drilling into the affected fulfillment rows isn't possible yet (the grid
 * ships in P3) — the affordance is visibly present but disabled, never a
 * dead link.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { EmptyState } from '@/shared/ui/empty-state'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type { DriftSummaryValue } from '@/shared/i18n/glossary'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [boolean] }>()

const { t } = useI18n({ useScope: 'global' })
const ctx = useWaveWorkspaceContext()

const signals = computed<dto.BasisDriftSignalDTO[]>(() => ctx.snapshot.value?.overview.basisDriftSignals ?? [])

// The known `basisKind` values across both the older single-word form
// ("supplier_order"/"shipment") and the newer `*_basis` suffixed form
// (internal/app/basis_drift_usecase.go emits both depending on code path) —
// anything else falls back to `waveWorkspace.drift.basisKind.unknown`.
const KNOWN_BASIS_KINDS = new Set([
  'supplier_order',
  'shipment',
  'supplier_order_basis',
  'shipment_basis',
  'channel_sync_basis',
])

// The known `driftReasonCodes` values (internal/app/basis_drift_usecase.go)
// — anything else falls back to `overview.drift.reasonCodes.unknown`.
const KNOWN_REASON_CODES = new Set([
  'target_deleted',
  'external_basis_stale',
  'projection_hash_unavailable',
  'projection_changed',
])

function basisKindLabel(basisKind: string): string {
  const key = KNOWN_BASIS_KINDS.has(basisKind) ? basisKind : 'unknown'
  return t('waveWorkspace.drift.basisKind.' + key)
}

function reasonCodeLabel(code: string): string {
  const key = KNOWN_REASON_CODES.has(code) ? code : 'unknown'
  return t('overview.drift.reasonCodes.' + key)
}

/** Per-signal driftSummary-compatible value — P2 foundations contract decision #3, NOT a new glossary dimension. */
function signalDriftSummary(signal: dto.BasisDriftSignalDTO): DriftSummaryValue {
  if (signal.basisDriftStatus === 'in_sync') return 'in_sync'
  if (signal.reviewRequirement === 'required') return 'drifted_required'
  if (signal.reviewRequirement === 'recommended') return 'drifted_recommended'
  return 'drifted_none'
}

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}
</script>

<template>
  <DetailDrawer :show="props.show" size="md" :title="t('waveWorkspace.drift.title')" @update:show="handleUpdateShow">
    <EmptyState v-if="signals.length === 0" size="sm" :title="t('taskCenter.actionStream.empty')" />
    <ul v-else class="wave-drift-drawer__list">
      <li v-for="(signal, index) in signals" :key="index" class="wave-drift-drawer__row">
        <div class="wave-drift-drawer__row-header">
          <span class="wave-drift-drawer__basis-kind">{{ basisKindLabel(signal.basisKind) }}</span>
          <StatusBadge dimension="driftSummary" :value="signalDriftSummary(signal)" size="sm" />
        </div>
        <span class="wave-drift-drawer__requirement">
          {{ t('waveWorkspace.drift.reviewRequirement.' + signal.reviewRequirement) }}
        </span>
        <div v-if="signal.driftReasonCodes.length > 0" class="wave-drift-drawer__reasons">
          <span v-for="code in signal.driftReasonCodes" :key="code" class="wave-drift-drawer__reason-chip">
            {{ reasonCodeLabel(code) }}
          </span>
        </div>
        <button
          type="button"
          class="wave-drift-drawer__drill"
          disabled
          :aria-label="t('waveWorkspace.drift.drillDisabledHint')"
        >
          {{ t('waveWorkspace.drift.drillDisabledHint') }}
        </button>
      </li>
    </ul>
  </DetailDrawer>
</template>

<style scoped>
.wave-drift-drawer__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.wave-drift-drawer__row {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--card-bg);
}

.wave-drift-drawer__row-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.wave-drift-drawer__basis-kind {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.wave-drift-drawer__requirement {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.wave-drift-drawer__reasons {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.wave-drift-drawer__reason-chip {
  display: inline-flex;
  align-items: center;
  padding: var(--statusbadge-padding-y) var(--statusbadge-padding-x);
  border-radius: var(--statusbadge-radius);
  border: 1px solid var(--status-neutral-border);
  background: var(--status-neutral-bg);
  color: var(--status-neutral-fg);
  font-family: var(--font-body);
  font-size: var(--statusbadge-font-size);
  font-weight: var(--statusbadge-font-weight);
}

.wave-drift-drawer__drill {
  align-self: flex-start;
  margin-top: var(--space-1);
  padding: var(--space-1) var(--space-2);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  cursor: not-allowed;
}
</style>
