<script setup lang="ts">
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSpin } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { PreviewRow, TemplatePreview } from '@/entities/models'

/**
 * TemplatePreviewPanel — renders one backend `TemplatePreview` (from
 * PreviewMapping or PreviewTemplate): the total / dropped row counters, the
 * semantic-row grid (one column per mapped key, localized titles, compact
 * fingerprint), and the parse-issue list. Shared by the Library template
 * editor / test drawer and the Inbox import modal so every "what will this
 * file become" surface looks the same.
 */
const props = withDefaults(
  defineProps<{
    preview: TemplatePreview | null
    loading?: boolean
    /** Already-resolved failure message; rendered instead of the grid. */
    error?: string
    /** Semantic dictionary order used to sort the value columns; unknown keys trail. */
    keyOrder?: string[]
  }>(),
  {
    loading: false,
    error: '',
    keyOrder: () => [],
  },
)

const { t, te } = useI18n({ useScope: 'global' })

function semanticLabel(key: string): string {
  const labelKey = `templateEditor.semanticKeys.${key}`
  return te(labelKey) ? t(labelKey) : key
}

const valueKeys = computed(() => {
  const seen = new Set<string>()
  for (const row of props.preview?.Rows ?? []) {
    for (const key of Object.keys(row.Values ?? {})) seen.add(key)
  }
  const rank = new Map(props.keyOrder.map((key, index) => [key, index]))
  return [...seen].sort((a, b) => {
    const ra = rank.get(a) ?? Number.MAX_SAFE_INTEGER
    const rb = rank.get(b) ?? Number.MAX_SAFE_INTEGER
    return ra === rb ? a.localeCompare(b) : ra - rb
  })
})

interface GridRow {
  __key: string
  row: PreviewRow
}

const gridRows = computed<GridRow[]>(() =>
  (props.preview?.Rows ?? []).map((row, index) => ({ __key: `${row.LineNo}-${index}`, row })),
)

const gridColumns = computed(() =>
  createColumns<GridRow>([
    {
      key: 'lineNo',
      title: t('templatePreview.lineNo'),
      type: 'text',
      width: 72,
      sortable: false,
      ellipsis: false,
      render: ({ row }) =>
        h(
          'span',
          {
            class: 'template-preview-panel__line',
            title:
              row.SourceRow && row.SourceRow !== row.LineNo
                ? t('templatePreview.sourceRow', { n: row.SourceRow })
                : undefined,
          },
          `#${row.LineNo}`,
        ),
    },
    ...valueKeys.value.map((key) => ({
      key: `value-${key}`,
      title: semanticLabel(key),
      type: 'text' as const,
      sortable: false,
      minWidth: 120,
      getValue: ({ row }: GridRow) => row.Values[key],
    })),
    {
      key: 'fingerprint',
      title: t('templatePreview.fingerprint'),
      type: 'text',
      width: 110,
      sortable: false,
      ellipsis: false,
      render: ({ row }) =>
        row.Fingerprint
          ? h(
              'code',
              { class: 'template-preview-panel__fingerprint', title: row.Fingerprint },
              row.Fingerprint.slice(0, 10),
            )
          : '—',
    },
  ]),
)

const shownCount = computed(() => props.preview?.Rows.length ?? 0)
const hasDropped = computed(() => (props.preview?.DroppedRows ?? 0) > 0)
</script>

<template>
  <div class="template-preview-panel">
    <NSpin :show="loading">
      <p v-if="error" class="template-preview-panel__error">
        {{ t('templatePreview.failed', { message: error }) }}
      </p>

      <template v-else-if="preview">
        <div class="template-preview-panel__stats">
          <span class="template-preview-panel__stat">
            {{ t('templatePreview.totalRows', { n: preview.TotalRows }) }}
          </span>
          <span
            class="template-preview-panel__stat"
            :class="{ 'template-preview-panel__stat--warning': hasDropped }"
          >
            {{ t('templatePreview.droppedRows', { n: preview.DroppedRows }) }}
          </span>
          <span class="template-preview-panel__stat template-preview-panel__stat--muted">
            {{ t('templatePreview.shownRows', { n: shownCount }) }}
          </span>
        </div>

        <DataGrid
          :columns="gridColumns"
          :rows="gridRows"
          row-key="__key"
          pagination="none"
          :empty="{ title: t('templatePreview.noRows') }"
        />

        <div class="template-preview-panel__issues">
          <h5 class="template-preview-panel__issues-title">
            {{ t('templatePreview.issuesTitle', { n: preview.Issues.length }) }}
          </h5>
          <p v-if="!preview.Issues.length" class="template-preview-panel__hint">
            {{ t('templatePreview.noIssues') }}
          </p>
          <ul v-else class="template-preview-panel__issue-list">
            <li
              v-for="(issue, index) in preview.Issues"
              :key="`${issue.LineNo}-${issue.Key}-${index}`"
              class="template-preview-panel__issue-row"
            >
              <span class="template-preview-panel__line">#{{ issue.LineNo }}</span>
              <span class="template-preview-panel__issue-key">{{ semanticLabel(issue.Key) }}</span>
              <span>{{ issue.Message }}</span>
            </li>
          </ul>
        </div>
      </template>

      <p v-else class="template-preview-panel__hint">
        <slot name="idle">{{ t('templatePreview.idle') }}</slot>
      </p>
    </NSpin>
  </div>
</template>

<style scoped>
.template-preview-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.template-preview-panel__stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  margin-bottom: var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.template-preview-panel__stat {
  font-variant-numeric: tabular-nums;
}

.template-preview-panel__stat--warning {
  color: var(--status-warning-fg);
  font-weight: var(--font-weight-medium);
}

.template-preview-panel__stat--muted {
  color: var(--color-text-muted);
}

.template-preview-panel__error {
  margin: 0;
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.template-preview-panel__hint {
  margin: 0;
  color: var(--color-text-muted);
  font-size: var(--font-size-sm);
}

.template-preview-panel__issues {
  margin-top: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.template-preview-panel__issues-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.template-preview-panel__issue-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.template-preview-panel__issue-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.template-preview-panel__issue-key {
  color: var(--color-text-secondary);
}
</style>

<style>
/* Grid cells render outside this SFC's scoped subtree (NDataTable `render`). */
.template-preview-panel__line {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  min-width: 40px;
  display: inline-block;
}

.template-preview-panel__fingerprint {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}
</style>
