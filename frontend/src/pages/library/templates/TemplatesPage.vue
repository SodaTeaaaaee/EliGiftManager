<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTag } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { useFeedback } from '@/shared/ui/feedback'
import {
  deleteTemplate,
  getDocumentTypeCatalog,
  getNamedTransformers,
  getSemanticDictionary,
  getTemplate,
  listPlatforms,
  listTemplates,
} from '@/shared/api/bridge'
import type { DocumentTypeInfo, Platform, TemplateConfig } from '@/entities/models'
import TemplateListSection from './TemplateListSection.vue'
import TemplateEditorDrawer from './TemplateEditorDrawer.vue'
import TemplateTestDrawer from './TemplateTestDrawer.vue'
import CarrierMappingSection from './CarrierMappingSection.vue'

/**
 * Library › 模板 route. Owns the shared reference data (templates, platforms,
 * semantic dictionary, document-type catalog) and hands it to the list, the
 * editor drawer, the test drawer, and the carrier-mapping section.
 */
const { t } = useI18n()
const feedback = useFeedback()

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const loading = ref(false)
const templates = ref<TemplateConfig[]>([])
const platforms = ref<Platform[]>([])
const dictionary = ref<string[]>([])
const transformers = ref<string[]>([])
const docTypes = ref<DocumentTypeInfo[]>([])

async function loadData(): Promise<void> {
  loading.value = true
  try {
    const [tmplRes, platRes, dictRes, transRes, typeRes] = await Promise.all([
      listTemplates(),
      listPlatforms(),
      getSemanticDictionary(),
      getNamedTransformers(),
      getDocumentTypeCatalog(),
    ])
    templates.value = tmplRes
    platforms.value = platRes
    dictionary.value = dictRes
    transformers.value = transRes
    docTypes.value = typeRes
  } catch (err) {
    feedback.error(t('templates.loadFailed'), errMsg(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

// ── Editor drawer (create blank / copy built-in / edit active) ──

const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editorSeed = ref<TemplateConfig | null>(null)

function openCreate(): void {
  editorMode.value = 'create'
  editorSeed.value = null
  editorOpen.value = true
}

function openCopy(builtin: TemplateConfig): void {
  editorMode.value = 'create'
  editorSeed.value = builtin
  editorOpen.value = true
}

/** Re-read the row before editing so a stale list never overwrites newer JSON. */
async function openEdit(template: TemplateConfig): Promise<void> {
  let fresh = template
  try {
    fresh = await getTemplate(template.ID)
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
    return
  }
  editorMode.value = 'edit'
  editorSeed.value = fresh
  editorOpen.value = true
}

async function handleSaved(): Promise<void> {
  await loadData()
}

// ── Test drawer ──

const testOpen = ref(false)
const testTemplate = ref<TemplateConfig | null>(null)

function openTest(template: TemplateConfig): void {
  testTemplate.value = template
  testOpen.value = true
}

// ── Delete ──

async function handleDelete(template: TemplateConfig): Promise<void> {
  try {
    await deleteTemplate(template.ID)
    feedback.success(t('templates.deleteSuccess'))
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  }
}
</script>

<template>
  <div class="templates-page">
    <TemplateListSection
      :templates="templates"
      :platforms="platforms"
      :doc-types="docTypes"
      :loading="loading"
      @create="openCreate"
      @copy="openCopy"
      @edit="openEdit"
      @test="openTest"
      @delete="handleDelete"
      @refresh="loadData"
    />

    <CarrierMappingSection :platforms="platforms" />

    <div class="templates-page__meta-grid">
      <SectionCard :title="t('library.semanticDictionary')">
        <div class="templates-page__tags">
          <NTag v-for="key in dictionary" :key="key" size="small" type="info">
            {{ key }}
          </NTag>
        </div>
      </SectionCard>

      <SectionCard :title="t('library.namedTransformers')">
        <div class="templates-page__tags">
          <NTag v-for="trans in transformers" :key="trans" size="small" type="success">
            {{ trans }}
          </NTag>
        </div>
      </SectionCard>
    </div>

    <TemplateEditorDrawer
      v-model:show="editorOpen"
      :mode="editorMode"
      :seed="editorSeed"
      :platforms="platforms"
      :doc-types="docTypes"
      :dictionary="dictionary"
      @saved="handleSaved"
    />

    <TemplateTestDrawer v-model:show="testOpen" :template="testTemplate" :dictionary="dictionary" />
  </div>
</template>

<style scoped>
.templates-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.templates-page__meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.templates-page__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
</style>
