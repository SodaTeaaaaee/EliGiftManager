<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { TemplatePreviewPanel } from '@/shared/ui/template-preview'
import { pickFile, previewTemplate } from '@/shared/api/bridge'
import type { TemplateConfig, TemplatePreview } from '@/entities/models'
import { useTemplateLabels } from './templateLabels'

/**
 * TemplateTestDrawer — 「测试」 for a saved or built-in template: pick a
 * sample file, run PreviewTemplate, and show the shared preview panel. Works
 * for built-ins too, which is how operators judge a catalog entry before
 * copying it into an active template.
 */
const props = defineProps<{
  show: boolean
  template: TemplateConfig | null
  dictionary: string[]
}>()

const emit = defineEmits<{
  'update:show': [boolean]
}>()

const { t } = useI18n()
const { documentTypeLabel } = useTemplateLabels()

const PREVIEW_LIMIT = 20
const sampleFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]

const filePath = ref('')
const preview = ref<TemplatePreview | null>(null)
const loading = ref(false)
const error = ref('')

watch(
  () => props.show,
  (open) => {
    if (!open) return
    filePath.value = ''
    preview.value = null
    error.value = ''
  },
)

async function run(): Promise<void> {
  if (!props.template || !filePath.value) return
  loading.value = true
  error.value = ''
  try {
    preview.value = await previewTemplate(props.template.ID, filePath.value, PREVIEW_LIMIT)
  } catch (err) {
    preview.value = null
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function handlePickFile(): Promise<void> {
  const path = await pickFile(sampleFileFilters)
  if (!path) return
  filePath.value = path
  await run()
}
</script>

<template>
  <DetailDrawer
    :show="show"
    :title="t('templates.testTitle')"
    size="xl"
    @update:show="(v) => emit('update:show', v)"
  >
    <div v-if="template" class="template-test__meta">
      <span class="template-test__name">{{ template.Name }}</span>
      <StatusBadge dimension="templateDirection" :value="template.Direction || 'input'" size="sm" />
      <span class="template-test__muted">{{ documentTypeLabel(template.DocumentType) }}</span>
      <span class="template-test__muted">{{ t('templates.versionTag', { n: template.Version }) }}</span>
      <span v-if="template.Builtin" class="template-test__muted">{{ t('templates.list.builtinTag') }}</span>
    </div>

    <div class="template-test__file">
      <NButton type="primary" size="small" :loading="loading" @click="handlePickFile">
        {{ filePath ? t('templates.changeSample') : t('templates.uploadSample') }}
      </NButton>
      <span v-if="filePath" class="template-test__path" :title="filePath">{{ filePath }}</span>
      <NButton v-if="filePath" size="small" secondary :loading="loading" @click="run">
        {{ t('templates.rerunTest') }}
      </NButton>
    </div>

    <TemplatePreviewPanel :preview="preview" :loading="loading" :error="error" :key-order="dictionary">
      <template #idle>{{ t('templates.testIdle') }}</template>
    </TemplatePreviewPanel>

    <template #footer>
      <NButton @click="emit('update:show', false)">{{ t('common.close') }}</NButton>
    </template>
  </DetailDrawer>
</template>

<style scoped>
.template-test__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.template-test__name {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.template-test__muted {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.template-test__file {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.template-test__path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1 1 200px;
  min-width: 0;
}
</style>
