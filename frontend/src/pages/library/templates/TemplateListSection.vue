<script setup lang="ts">
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NPopconfirm, NSpace, NTag } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import { parseMappingRules } from '@/shared/ui/field-mapping'
import type { DocumentTypeInfo, Platform, TemplateConfig } from '@/entities/models'
import { parseLayoutConfig } from './layoutConfig'
import { inEffectTemplateIDs } from './templateCatalog'
import { useTemplateLabels } from './templateLabels'

/**
 * TemplateListSection — the two list areas of the Library templates page:
 * 「活跃模板」 (editable, testable, deletable, with the 当前生效 badge for the
 * document types the backend auto-picks on export) and the read-only
 * 「内置模板」 catalog (copy into an active template, or test against a file).
 */
const props = defineProps<{
  templates: TemplateConfig[]
  platforms: Platform[]
  docTypes: DocumentTypeInfo[]
  loading: boolean
}>()

const emit = defineEmits<{
  create: []
  edit: [TemplateConfig]
  copy: [TemplateConfig]
  test: [TemplateConfig]
  delete: [TemplateConfig]
  refresh: []
}>()

const { t } = useI18n()
const { documentTypeLabel } = useTemplateLabels()

const activeTemplates = computed(() => props.templates.filter((tpl) => !tpl.Builtin))
const builtinTemplates = computed(() => props.templates.filter((tpl) => tpl.Builtin))

/** Output document types are auto-picked on export, so only they carry 「当前生效」. */
const autoPickedTypes = computed(
  () => new Set(props.docTypes.filter((info) => info.Direction === 'output').map((info) => info.Key)),
)
const inEffect = computed(() => inEffectTemplateIDs(props.templates, autoPickedTypes.value))

function platformName(id: number): string {
  const platform = props.platforms.find((p) => p.ID === id)
  return platform ? platform.Name : t('templates.list.unknownPlatform', { id })
}

function configSummary(row: TemplateConfig): string {
  if (row.Direction === 'output') {
    const layout = parseLayoutConfig(row.LayoutJSON)
    return t('templates.layoutSummary', {
      format: layout.format.toUpperCase(),
      count: layout.columns.length,
    })
  }
  const mapping = parseMappingRules(row.MappingJSON)
  const count =
    mapping.mode === 'positional'
      ? Object.keys(mapping.positions ?? {}).length
      : Object.keys(mapping.columns).length
  const modeLabel =
    mapping.mode === 'positional'
      ? t('templateEditor.modePositional')
      : t('templateEditor.modeHeader')
  return t('templates.mappingSummary', { mode: modeLabel, count })
}

function renderName(row: TemplateConfig) {
  const children = [h('span', { class: 'template-list__name' }, row.Name)]
  if (inEffect.value.has(row.ID)) {
    children.push(
      h(
        NTag,
        { size: 'small', type: 'success', round: true, class: 'template-list__effective' },
        { default: () => t('templates.list.inEffect') },
      ),
    )
  }
  return h('span', { class: 'template-list__name-cell' }, children)
}

const sharedColumns: DataGridColumnSpec<TemplateConfig>[] = [
  {
    key: 'PlatformID',
    title: t('templates.editor.platform'),
    type: 'text',
    width: 150,
    getValue: (row) => platformName(row.PlatformID),
  },
  {
    key: 'DocumentType',
    title: t('library.documentType'),
    type: 'text',
    width: 130,
    getValue: (row) => documentTypeLabel(row.DocumentType),
  },
  {
    key: 'Direction',
    title: t('library.direction'),
    type: 'status',
    dimension: 'templateDirection',
    width: 110,
    getValue: (row) => row.Direction || 'input',
  },
  {
    key: 'summary',
    title: t('templates.list.configSummary'),
    type: 'text',
    width: 160,
    sortable: false,
    getValue: (row) => configSummary(row),
  },
]

const activeColumns = computed(() =>
  createColumns<TemplateConfig>([
    {
      key: 'Name',
      title: t('library.templateName'),
      type: 'text',
      minWidth: 200,
      render: renderName,
    },
    ...sharedColumns,
    {
      key: 'Version',
      title: t('library.version'),
      type: 'text',
      width: 70,
      getValue: (row) => t('templates.versionTag', { n: row.Version }),
    },
    {
      key: 'UpdatedAt',
      title: t('templates.list.updatedAt'),
      type: 'date',
      format: 'datetime',
      width: 160,
    },
    {
      key: 'Notes',
      title: t('library.notes'),
      type: 'text',
      minWidth: 140,
      sortable: false,
    },
    {
      key: 'actions',
      title: t('common.actions'),
      type: 'actions',
      width: 190,
      render: (row) =>
        h(NSpace, { size: 'small', justify: 'end', wrap: false }, () => [
          h(
            NButton,
            { size: 'tiny', type: 'primary', secondary: true, onClick: () => emit('edit', row) },
            { default: () => t('common.edit') },
          ),
          h(
            NButton,
            { size: 'tiny', secondary: true, onClick: () => emit('test', row) },
            { default: () => t('templates.test') },
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => emit('delete', row),
              positiveText: t('templates.list.deleteAction'),
              negativeText: t('common.cancel'),
            },
            {
              trigger: () =>
                h(
                  NButton,
                  { size: 'tiny', type: 'error', quaternary: true },
                  { default: () => t('templates.list.deleteAction') },
                ),
              default: () => t('templates.list.deleteConfirm', { name: row.Name }),
            },
          ),
        ]),
    },
  ]),
)

const builtinColumns = computed(() =>
  createColumns<TemplateConfig>([
    {
      key: 'Name',
      title: t('library.templateName'),
      type: 'text',
      minWidth: 200,
    },
    ...sharedColumns,
    {
      key: 'Notes',
      title: t('library.notes'),
      type: 'text',
      minWidth: 160,
      sortable: false,
    },
    {
      key: 'actions',
      title: t('common.actions'),
      type: 'actions',
      width: 210,
      render: (row) =>
        h(NSpace, { size: 'small', justify: 'end', wrap: false }, () => [
          h(
            NButton,
            { size: 'tiny', type: 'primary', secondary: true, onClick: () => emit('copy', row) },
            { default: () => t('templates.list.copyToActive') },
          ),
          h(
            NButton,
            { size: 'tiny', secondary: true, onClick: () => emit('test', row) },
            { default: () => t('templates.test') },
          ),
        ]),
    },
  ]),
)
</script>

<template>
  <div class="template-list">
    <SectionCard :title="t('templates.list.activeTitle')" :description="t('templates.list.activeDesc')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="emit('create')">
            {{ t('library.createTemplate') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="emit('refresh')">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <div v-if="!loading && !activeTemplates.length" class="template-list__empty">
        <EmptyState
          :title="t('templates.list.emptyActiveTitle')"
          :description="t('templates.list.emptyActiveDesc')"
          size="sm"
        >
          <NButton size="small" type="primary" @click="emit('create')">
            {{ t('library.createTemplate') }}
          </NButton>
        </EmptyState>
      </div>
      <DataGrid
        v-else
        :columns="activeColumns"
        :rows="activeTemplates"
        row-key="ID"
        :loading="loading"
        pagination="none"
      />
    </SectionCard>

    <SectionCard :title="t('templates.list.builtinTitle')" :description="t('templates.list.builtinDesc')">
      <div v-if="!loading && !builtinTemplates.length" class="template-list__empty">
        <EmptyState :title="t('templates.list.emptyBuiltin')" size="sm" />
      </div>
      <DataGrid
        v-else
        :columns="builtinColumns"
        :rows="builtinTemplates"
        row-key="ID"
        :loading="loading"
        pagination="none"
      />
    </SectionCard>
  </div>
</template>

<style scoped>
.template-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.template-list__empty {
  padding: var(--space-2) 0;
}
</style>

<style>
/* Cells render through NDataTable `render`, outside this SFC's scoped subtree. */
.template-list__name-cell {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.template-list__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-list__effective {
  flex-shrink: 0;
}
</style>
