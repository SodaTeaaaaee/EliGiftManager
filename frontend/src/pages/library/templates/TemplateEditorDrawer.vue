<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NSelect } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import { TemplatePreviewPanel } from '@/shared/ui/template-preview'
import {
  FieldMappingEditor,
  emptyFieldMapping,
  parseMappingRules,
  serializeMappingRules,
  type FieldMappingValue,
} from '@/shared/ui/field-mapping'
import { createTemplate, previewMapping, updateTemplate } from '@/shared/api/bridge'
import type {
  DocumentTypeInfo,
  Platform,
  SampleFileInfo,
  TemplateConfig,
  TemplatePreview,
} from '@/entities/models'
import SampleFilePanel from './SampleFilePanel.vue'
import LayoutEditor from './LayoutEditor.vue'
import {
  emptyLayoutConfig,
  parseLayoutConfig,
  serializeLayoutConfig,
  type LayoutConfigValue,
} from './layoutConfig'
import { buildSampleEditorRows, defaultVisibleKeysFor } from './templateCatalog'
import { useTemplateLabels } from './templateLabels'

/**
 * TemplateEditorDrawer — the create / edit workbench for one template.
 * Sections: 归属 (document type → locked direction → kind-filtered platform →
 * name → notes), 样例文件 + 字段映射 + 验证 for input types, 输出布局 for output
 * types. `seed` is the template being edited, or a built-in to copy from in
 * create mode (copying prefills every field and saves as a fresh active
 * template through CreateTemplate).
 */
const props = defineProps<{
  show: boolean
  mode: 'create' | 'edit'
  seed: TemplateConfig | null
  platforms: Platform[]
  docTypes: DocumentTypeInfo[]
  dictionary: string[]
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  saved: [TemplateConfig]
}>()

const { t } = useI18n()
const feedback = useFeedback()
const { documentTypeLabel, groups, destFieldsFor } = useTemplateLabels()

const PREVIEW_LIMIT = 20

// ── Form state ──

const documentType = ref('')
const platformId = ref<number | null>(null)
const name = ref('')
const notes = ref('')
const mapping = ref<FieldMappingValue>(emptyFieldMapping('header'))
const layout = ref<LayoutConfigValue>(emptyLayoutConfig())

const sampleFilePath = ref('')
const sampleInfo = ref<SampleFileInfo | null>(null)

const preview = ref<TemplatePreview | null>(null)
const previewLoading = ref(false)
const previewError = ref('')

const saving = ref(false)

const isEdit = computed(() => props.mode === 'edit' && props.seed != null)
const isCopy = computed(() => props.mode === 'create' && props.seed != null)

const docTypeInfo = computed(() => props.docTypes.find((info) => info.Key === documentType.value) ?? null)
const direction = computed(() => docTypeInfo.value?.Direction ?? props.seed?.Direction ?? 'input')
const isInput = computed(() => direction.value === 'input')

const documentTypeOptions = computed(() =>
  props.docTypes.map((info) => ({ label: documentTypeLabel(info.Key), value: info.Key })),
)

const platformOptions = computed(() => {
  const kind = docTypeInfo.value?.PlatformKind
  return props.platforms
    .filter((platform) => !kind || platform.Kind === kind)
    .map((platform) => ({ label: `${platform.Name} (${platform.Key})`, value: platform.ID }))
})

const platformKindLabel = computed(() => {
  const kind = docTypeInfo.value?.PlatformKind
  return kind ? t(`glossary.platformKind.${kind}.label`) : ''
})

const title = computed(() => {
  if (isEdit.value) return t('templates.editor.editTitle', { name: props.seed?.Name ?? '' })
  if (isCopy.value) return t('templates.editor.copyTitle', { name: props.seed?.Name ?? '' })
  return t('templates.editor.createTitle')
})

function resetFromSeed(): void {
  const seed = props.seed
  const firstType = props.docTypes.find((info) => info.Direction === 'input') ?? props.docTypes[0]
  documentType.value = seed?.DocumentType ?? firstType?.Key ?? ''
  platformId.value = seed?.PlatformID ?? null
  name.value = seed
    ? isEdit.value
      ? seed.Name
      : t('templates.editor.copyNameSuffix', { name: seed.Name })
    : ''
  notes.value = seed?.Notes ?? ''
  mapping.value = seed ? parseMappingRules(seed.MappingJSON) : emptyFieldMapping('header')
  layout.value = seed ? parseLayoutConfig(seed.LayoutJSON) : emptyLayoutConfig()
  sampleFilePath.value = ''
  sampleInfo.value = null
  preview.value = null
  previewError.value = ''
  ensurePlatformMatchesKind()
}

/** Keep the platform inside the kind the chosen document type allows. */
function ensurePlatformMatchesKind(): void {
  const allowed = platformOptions.value.map((option) => option.value)
  if (platformId.value != null && allowed.includes(platformId.value)) return
  platformId.value = allowed[0] ?? null
}

watch(
  () => props.show,
  (open) => {
    if (open) resetFromSeed()
  },
  { immediate: true },
)

function handleDocumentTypeChange(next: string): void {
  documentType.value = next
  ensurePlatformMatchesKind()
  preview.value = null
}

// ── Sample file ↔ mapping mode ──

const hasHeader = computed(() => mapping.value.mode !== 'positional')

function setHasHeader(next: boolean): void {
  mapping.value = {
    ...mapping.value,
    mode: next ? 'header' : 'positional',
    hasHeader: next,
  }
  preview.value = null
}

function setSheetName(next: string): void {
  mapping.value = { ...mapping.value, sheetName: next.trim() !== '' ? next : undefined }
  preview.value = null
}

function handleSampleInspected(filePath: string, info: SampleFileInfo): void {
  sampleFilePath.value = filePath
  sampleInfo.value = info
  preview.value = null
  previewError.value = ''
}

function handleSampleCleared(): void {
  sampleFilePath.value = ''
  sampleInfo.value = null
  preview.value = null
  previewError.value = ''
}

const editorRows = computed(() => buildSampleEditorRows(sampleInfo.value, hasHeader.value))

const destFields = computed(() => destFieldsFor(props.dictionary))
const defaultVisibleKeys = computed(() => defaultVisibleKeysFor(documentType.value, props.dictionary))

function handleMappingChange(next: FieldMappingValue): void {
  mapping.value = next
  preview.value = null
}

// ── Validation preview (PreviewMapping) ──

async function runPreview(): Promise<void> {
  if (!sampleFilePath.value) return
  previewLoading.value = true
  previewError.value = ''
  try {
    preview.value = await previewMapping(
      serializeMappingRules(mapping.value),
      documentType.value,
      sampleFilePath.value,
      PREVIEW_LIMIT,
    )
  } catch (err) {
    preview.value = null
    previewError.value = err instanceof Error ? err.message : String(err)
  } finally {
    previewLoading.value = false
  }
}

// ── Save ──

const canSave = computed(() => {
  if (!platformId.value || name.value.trim() === '' || !docTypeInfo.value) return false
  if (!isInput.value && layout.value.columns.length === 0) return false
  return true
})

async function handleSave(): Promise<void> {
  if (!canSave.value || !platformId.value) return
  saving.value = true
  try {
    const payload = {
      PlatformID: platformId.value,
      DocumentType: documentType.value,
      Direction: direction.value,
      Name: name.value.trim(),
      Notes: notes.value.trim(),
      MappingJSON: isInput.value ? serializeMappingRules(mapping.value) : props.seed?.MappingJSON ?? '',
      LayoutJSON: !isInput.value ? serializeLayoutConfig(layout.value) : props.seed?.LayoutJSON ?? '',
      ExtraData: props.seed?.ExtraData ?? '',
    }
    const saved =
      isEdit.value && props.seed
        ? await updateTemplate({ ...payload, ID: props.seed.ID, Version: props.seed.Version, Builtin: false })
        : await createTemplate(payload)
    feedback.success(t('library.templateSuccess'))
    emit('saved', saved)
    emit('update:show', false)
  } catch (err) {
    feedback.error(t('templates.editor.saveFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <DetailDrawer :show="show" :title="title" size="xl" @update:show="(v) => emit('update:show', v)">
    <section class="template-editor__section">
      <header class="template-editor__section-head">
        <h4 class="template-editor__section-title">{{ t('templates.editor.sectionOwnership') }}</h4>
        <p class="template-editor__section-hint">{{ t('templates.editor.sectionOwnershipHint') }}</p>
      </header>
      <p v-if="!docTypes.length" class="template-editor__warning">
        {{ t('templates.editor.catalogUnavailable') }}
      </p>
      <NForm label-placement="left" label-width="110" class="template-editor__form">
        <NFormItem :label="t('library.documentType')">
          <div class="template-editor__field">
            <NSelect
              class="template-editor__field-control"
              :value="documentType || null"
              :options="documentTypeOptions"
              :disabled="isEdit || !docTypes.length"
              :placeholder="t('common.pleaseSelect')"
              @update:value="(v) => handleDocumentTypeChange(String(v ?? ''))"
            />
            <span v-if="isEdit" class="template-editor__locked">{{ t('templates.editor.locked') }}</span>
          </div>
        </NFormItem>
        <NFormItem :label="t('library.direction')">
          <div class="template-editor__field">
            <StatusBadge dimension="templateDirection" :value="direction" />
            <span class="template-editor__locked">{{ t('templates.editor.directionLocked') }}</span>
          </div>
        </NFormItem>
        <NFormItem :label="t('templates.editor.platform')">
          <div class="template-editor__field">
            <NSelect
              v-model:value="platformId"
              class="template-editor__field-control"
              :options="platformOptions"
              :disabled="isEdit || !platformOptions.length"
              :placeholder="
                platformOptions.length
                  ? t('common.pleaseSelect')
                  : t('templates.editor.noPlatformForKind', { kind: platformKindLabel })
              "
            />
            <span v-if="platformKindLabel && !isEdit" class="template-editor__locked">
              {{ t('templates.editor.platformKindHint', { kind: platformKindLabel }) }}
            </span>
          </div>
        </NFormItem>
        <NFormItem :label="t('library.templateName')">
          <NInput v-model:value="name" :placeholder="t('templates.editor.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput
            v-model:value="notes"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            :placeholder="t('templates.editor.notesPlaceholder')"
          />
        </NFormItem>
      </NForm>
    </section>

    <template v-if="isInput">
      <section class="template-editor__section">
        <header class="template-editor__section-head">
          <h4 class="template-editor__section-title">{{ t('templates.editor.sectionSample') }}</h4>
          <p class="template-editor__section-hint">{{ t('templates.editor.sectionSampleHint') }}</p>
        </header>
        <SampleFilePanel
          :file-path="sampleFilePath"
          :sample="sampleInfo"
          :sheet-name="mapping.sheetName ?? ''"
          :has-header="hasHeader"
          @inspected="handleSampleInspected"
          @cleared="handleSampleCleared"
          @update:sheet-name="setSheetName"
          @update:has-header="setHasHeader"
        />
      </section>

      <section class="template-editor__section">
        <header class="template-editor__section-head">
          <h4 class="template-editor__section-title">{{ t('templates.editor.sectionMapping') }}</h4>
          <p class="template-editor__section-hint">{{ t('templates.editor.sectionMappingHint') }}</p>
        </header>
        <FieldMappingEditor
          :model-value="mapping"
          :dest-fields="destFields"
          :groups="groups"
          :default-visible-keys="defaultVisibleKeys"
          :source-headers="editorRows.sourceHeaders"
          :sample-rows="editorRows.sampleRows"
          hide-source-meta
          @update:model-value="handleMappingChange"
        />
      </section>

      <section class="template-editor__section">
        <header class="template-editor__section-head">
          <h4 class="template-editor__section-title">{{ t('templates.editor.sectionValidate') }}</h4>
          <p class="template-editor__section-hint">{{ t('templates.editor.sectionValidateHint') }}</p>
        </header>
        <div class="template-editor__validate-bar">
          <NButton
            size="small"
            type="primary"
            secondary
            :disabled="!sampleFilePath"
            :loading="previewLoading"
            @click="runPreview"
          >
            {{ t('templates.editor.runPreview') }}
          </NButton>
          <span v-if="!sampleFilePath" class="template-editor__locked">
            {{ t('templates.editor.previewNeedsSample') }}
          </span>
        </div>
        <TemplatePreviewPanel
          :preview="preview"
          :loading="previewLoading"
          :error="previewError"
          :key-order="dictionary"
        >
          <template #idle>{{ t('templates.editor.previewIdle') }}</template>
        </TemplatePreviewPanel>
      </section>
    </template>

    <section v-else class="template-editor__section">
      <header class="template-editor__section-head">
        <h4 class="template-editor__section-title">{{ t('templates.editor.sectionLayout') }}</h4>
        <p class="template-editor__section-hint">{{ t('templates.editor.sectionLayoutHint') }}</p>
      </header>
      <LayoutEditor v-model="layout" :dest-fields="destFields" :groups="groups" />
    </section>

    <template #footer>
      <NButton :disabled="saving" @click="emit('update:show', false)">{{ t('common.cancel') }}</NButton>
      <NButton type="primary" :loading="saving" :disabled="!canSave" @click="handleSave">
        {{ isEdit ? t('templates.editor.saveUpdate') : t('templates.editor.saveCreate') }}
      </NButton>
    </template>
  </DetailDrawer>
</template>

<style scoped>
.template-editor__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-bottom: var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.template-editor__section:last-of-type {
  border-bottom: none;
  padding-bottom: 0;
}

.template-editor__section-head {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.template-editor__section-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.template-editor__section-hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.template-editor__form {
  max-width: 720px;
}

.template-editor__field {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  min-width: 0;
}

.template-editor__field-control {
  flex: 1;
  min-width: 0;
}

.template-editor__locked {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.template-editor__warning {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
  font-size: var(--font-size-sm);
}

.template-editor__validate-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
</style>
