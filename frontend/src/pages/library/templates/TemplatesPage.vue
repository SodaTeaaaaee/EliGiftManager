<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import {
  createCarrierMapping,
  createTemplate,
  getNamedTransformers,
  getSemanticDictionary,
  listCarrierMappings,
  listPlatforms,
  listTemplates,
} from '@/shared/api/bridge'
import type { CarrierMapping, Platform, TemplateConfig } from '@/entities/models'

const { t } = useI18n()

const loading = ref(false)
const actionLoading = ref(false)
const templates = ref<TemplateConfig[]>([])
const platforms = ref<Platform[]>([])
const dictionary = ref<string[]>([])
const transformers = ref<string[]>([])

const selectedPlatformForCarriers = ref<number | null>(null)
const carrierMappings = ref<CarrierMapping[]>([])

const showCreateTemplateModal = ref(false)
const templateForm = ref({
  platformId: null as number | null,
  documentType: 'import_membership',
  direction: 'input',
  name: '',
  version: 1,
  mappingJson: '{}',
  layoutJson: '{}',
  notes: '',
})

const showCreateCarrierModal = ref(false)
const carrierForm = ref({
  externalCode: '',
  internalCode: 'SF',
  internalName: '顺丰速运',
})

async function loadData() {
  loading.value = true
  try {
    const [tmplRes, platRes, dictRes, transRes] = await Promise.all([
      listTemplates(),
      listPlatforms(),
      getSemanticDictionary(),
      getNamedTransformers(),
    ])
    templates.value = tmplRes
    platforms.value = platRes
    dictionary.value = dictRes
    transformers.value = transRes
    if (platRes.length > 0 && !selectedPlatformForCarriers.value) {
      selectedPlatformForCarriers.value = platRes[0].ID
    }
  } catch (err) {
    console.error('Failed to load templates data:', err)
  } finally {
    loading.value = false
  }
}

async function loadCarrierMappings() {
  if (!selectedPlatformForCarriers.value) return
  try {
    carrierMappings.value = await listCarrierMappings(selectedPlatformForCarriers.value)
  } catch (err) {
    console.error('Failed to load carrier mappings:', err)
  }
}

watch(selectedPlatformForCarriers, () => {
  void loadCarrierMappings()
})

onMounted(async () => {
  await loadData()
  await loadCarrierMappings()
})

const platformOptions = computed(() =>
  platforms.value.map((p) => ({
    label: `${p.Name} (${p.Key})`,
    value: p.ID,
  })),
)

function openCreateTemplate() {
  templateForm.value = {
    platformId: platformOptions.value[0]?.value ?? null,
    documentType: 'import_membership',
    direction: 'input',
    name: '',
    version: 1,
    mappingJson: '{}',
    layoutJson: '{}',
    notes: '',
  }
  showCreateTemplateModal.value = true
}

async function handleSaveTemplate() {
  if (!templateForm.value.platformId || !templateForm.value.name.trim()) return
  actionLoading.value = true
  try {
    await createTemplate({
      PlatformID: templateForm.value.platformId,
      DocumentType: templateForm.value.documentType,
      Direction: templateForm.value.direction,
      Name: templateForm.value.name.trim(),
      Version: templateForm.value.version,
      MappingJSON: templateForm.value.mappingJson,
      LayoutJSON: templateForm.value.layoutJson,
      Notes: templateForm.value.notes.trim(),
    })
    showCreateTemplateModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to save template:', err)
  } finally {
    actionLoading.value = false
  }
}

function openCreateCarrier() {
  carrierForm.value = {
    externalCode: '',
    internalCode: 'SF',
    internalName: '顺丰速运',
  }
  showCreateCarrierModal.value = true
}

async function handleSaveCarrier() {
  if (!selectedPlatformForCarriers.value || !carrierForm.value.externalCode.trim()) return
  actionLoading.value = true
  try {
    await createCarrierMapping({
      PlatformID: selectedPlatformForCarriers.value,
      ExternalCode: carrierForm.value.externalCode.trim(),
      InternalCode: carrierForm.value.internalCode.trim(),
      InternalName: carrierForm.value.internalName.trim(),
    })
    showCreateCarrierModal.value = false
    await loadCarrierMappings()
  } catch (err) {
    console.error('Failed to save carrier mapping:', err)
  } finally {
    actionLoading.value = false
  }
}

const templateColumns = [
  {
    title: '#',
    key: 'ID',
    width: 70,
    render(row: TemplateConfig) {
      return `#${row.ID}`
    },
  },
  {
    title: t('library.templateName'),
    key: 'Name',
  },
  {
    title: t('library.factoryPlatform'),
    key: 'PlatformID',
    render(row: TemplateConfig) {
      const plat = platforms.value.find((p) => p.ID === row.PlatformID)
      return plat ? plat.Name : `Platform #${row.PlatformID}`
    },
  },
  {
    title: t('library.documentType'),
    key: 'DocumentType',
  },
  {
    title: t('library.direction'),
    key: 'Direction',
    width: 120,
    render(row: TemplateConfig) {
      return h(StatusBadge, {
        dimension: 'templateDirection',
        value: row.Direction || 'input',
      })
    },
  },
  {
    title: t('library.version'),
    key: 'Version',
    width: 90,
    render(row: TemplateConfig) {
      return `v${row.Version}`
    },
  },
]

const carrierColumns = [
  {
    title: t('library.externalCode'),
    key: 'ExternalCode',
  },
  {
    title: t('library.internalCode'),
    key: 'InternalCode',
  },
  {
    title: t('library.internalName'),
    key: 'InternalName',
  },
]
</script>

<template>
  <div class="templates-page">
    <!-- Templates List -->
    <SectionCard :title="t('library.templatesTab')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateTemplate">
            {{ t('library.createTemplate') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!templates.length" class="templates-page__empty">
          <EmptyState :title="t('library.emptyTemplates')" size="sm" />
        </div>
        <div v-else class="templates-page__table">
          <NDataTable
            :columns="templateColumns"
            :data="templates"
            :row-key="(row: TemplateConfig) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Carrier Mappings Section -->
    <SectionCard :title="t('library.carrierMappings')">
      <template #actions>
        <NSpace align="center">
          <NSelect
            v-model:value="selectedPlatformForCarriers"
            :options="platformOptions"
            size="small"
            style="width: 180px"
          />
          <NButton size="small" type="primary" @click="openCreateCarrier">
            {{ t('library.createCarrierMapping') }}
          </NButton>
        </NSpace>
      </template>

      <div v-if="!carrierMappings.length" class="templates-page__empty">
        <EmptyState :title="t('library.carrierMappings')" size="sm" />
      </div>
      <div v-else class="templates-page__table">
        <NDataTable
          :columns="carrierColumns"
          :data="carrierMappings"
          :row-key="(row: CarrierMapping) => row.ID"
          size="small"
        />
      </div>
    </SectionCard>

    <!-- Semantic Dictionary & Named Transformers -->
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

    <!-- Create Template Modal -->
    <NModal
      v-model:show="showCreateTemplateModal"
      preset="card"
      :title="t('library.createTemplate')"
      style="width: 520px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.factoryPlatform')">
          <NSelect v-model:value="templateForm.platformId" :options="platformOptions" />
        </NFormItem>
        <NFormItem :label="t('library.templateName')">
          <NInput v-model:value="templateForm.name" />
        </NFormItem>
        <NFormItem :label="t('library.documentType')">
          <NInput v-model:value="templateForm.documentType" />
        </NFormItem>
        <NFormItem :label="t('library.direction')">
          <NRadioGroup v-model:value="templateForm.direction">
            <NSpace>
              <NRadio value="input">{{ t('glossary.templateDirection.input.label') }}</NRadio>
              <NRadio value="output">{{ t('glossary.templateDirection.output.label') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput v-model:value="templateForm.notes" type="textarea" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateTemplateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!templateForm.platformId || !templateForm.name.trim()"
            @click="handleSaveTemplate"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Create Carrier Modal -->
    <NModal
      v-model:show="showCreateCarrierModal"
      preset="card"
      :title="t('library.createCarrierMapping')"
      style="width: 480px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.externalCode')">
          <NInput v-model:value="carrierForm.externalCode" />
        </NFormItem>
        <NFormItem :label="t('library.internalCode')">
          <NInput v-model:value="carrierForm.internalCode" />
        </NFormItem>
        <NFormItem :label="t('library.internalName')">
          <NInput v-model:value="carrierForm.internalName" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateCarrierModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!carrierForm.externalCode.trim()"
            @click="handleSaveCarrier"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.templates-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.templates-page__empty {
  padding: var(--space-4) 0;
}

.templates-page__table {
  width: 100%;
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
