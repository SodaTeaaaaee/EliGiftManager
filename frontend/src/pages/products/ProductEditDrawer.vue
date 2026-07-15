<script setup lang="ts">
/**
 * ProductEditDrawer — create/edit form for one `ProductMaster` (plan §3.7
 * first half). Archiving is NOT a separate endpoint — `updateProductMaster`
 * is a full-object PUT, so an edit-mode save always re-sends every field
 * (see `internal/app/product_usecase.go`'s `UpdateProductMaster`).
 *
 * Named "Drawer" per the P6 fileset contract but implemented as an `NModal`
 * card, matching every other create/edit form in this tree (`CreateWaveDialog.vue`,
 * `RuleEditor.vue`) — no `NDrawer` precedent exists for a form this small.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NInput, NSelect, NSwitch, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary, type ProductKindValue } from '@/shared/i18n/glossary'
import { createProductMaster, updateProductMaster } from '@/shared/api/bridge'
import { localImageUrl, parseDetailImagePaths, type ProductMaster } from '@/entities/product'

const props = defineProps<{
  show: boolean
  /** `null` = create mode; otherwise the master being edited. */
  master: ProductMaster | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  saved: [ProductMaster]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()

const PRODUCT_KINDS: ProductKindValue[] = ['badge', 'standee', 'charm', 'postcard', 'print', 'bundle', 'other']

const isEdit = computed(() => props.master !== null)

const supplierPlatform = ref('')
const factorySku = ref('')
const supplierProductRef = ref('')
const name = ref('')
const productKind = ref<ProductKindValue>('other')
const archived = ref(false)
const submitting = ref(false)

function resetForm(): void {
  const master = props.master
  supplierPlatform.value = master?.supplierPlatform ?? ''
  factorySku.value = master?.factorySku ?? ''
  supplierProductRef.value = master?.supplierProductRef ?? ''
  name.value = master?.name ?? ''
  productKind.value = (master?.productKind as ProductKindValue) ?? 'other'
  archived.value = master?.archived ?? false
}

// This dialog stays mounted (no `v-if` at the call site) — reset on every open.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const productKindOptions = computed<SelectOption[]>(() =>
  PRODUCT_KINDS.map((kind) => ({ label: glossaryLabel('productKind', kind), value: kind })),
)

/** Read-only cover URL for edit-mode preview (empty when none / create mode). */
const coverPreviewSrc = computed(() => localImageUrl(props.master?.coverImagePath))

/** Read-only detail thumbnail URLs for edit-mode preview. */
const detailPreviewSrcs = computed(() =>
  parseDetailImagePaths(props.master?.detailImagePaths).map((rel) => localImageUrl(rel)).filter(Boolean),
)

const canSubmit = computed(
  () =>
    !submitting.value &&
    supplierPlatform.value.trim().length > 0 &&
    factorySku.value.trim().length > 0 &&
    name.value.trim().length > 0,
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    const input = {
      supplierPlatform: supplierPlatform.value.trim(),
      factorySku: factorySku.value.trim(),
      supplierProductRef: supplierProductRef.value.trim(),
      name: name.value.trim(),
      productKind: productKind.value,
      // Images are catalog/ZIP-managed; preserve existing paths on manual edit.
      coverImagePath: props.master?.coverImagePath ?? '',
      detailImagePaths: props.master?.detailImagePaths ?? '',
      extraData: props.master?.extraData ?? '',
    }
    const saved =
      isEdit.value && props.master
        ? await updateProductMaster({ id: props.master.id, ...input, archived: archived.value })
        : await createProductMaster(input)
    emit('saved', saved)
    close()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t(isEdit ? 'products.editDialog.editTitle' : 'products.editDialog.createTitle')"
    :style="{ width: 'min(480px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('products.editDialog.nameLabel')">
        <NInput v-model:value="name" :disabled="submitting" @keydown.enter.prevent="handleSubmit" />
      </NFormItem>
      <NFormItem :label="t('products.editDialog.supplierPlatformLabel')">
        <NInput v-model:value="supplierPlatform" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('products.editDialog.factorySkuLabel')">
        <NInput v-model:value="factorySku" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('products.editDialog.supplierProductRefLabel')">
        <NInput v-model:value="supplierProductRef" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('products.editDialog.productKindLabel')">
        <NSelect v-model:value="productKind" :options="productKindOptions" :disabled="submitting" />
      </NFormItem>
      <NFormItem v-if="isEdit" :label="t('products.editDialog.archivedLabel')">
        <NSwitch v-model:value="archived" :disabled="submitting" />
      </NFormItem>
      <NFormItem v-if="isEdit" :label="t('products.editDialog.imagesLabel')">
        <div class="product-edit-drawer__images">
          <div class="product-edit-drawer__cover-block">
            <span class="product-edit-drawer__images-caption">{{ t('products.editDialog.coverLabel') }}</span>
            <img
              v-if="coverPreviewSrc"
              class="product-edit-drawer__thumb product-edit-drawer__thumb--cover"
              :src="coverPreviewSrc"
              :alt="t('products.editDialog.coverLabel')"
            />
            <div
              v-else
              class="product-edit-drawer__thumb product-edit-drawer__thumb--cover product-edit-drawer__thumb--placeholder"
            >
              {{ t('products.coverPlaceholder') }}
            </div>
          </div>
          <div v-if="detailPreviewSrcs.length > 0" class="product-edit-drawer__detail-block">
            <span class="product-edit-drawer__images-caption">{{ t('products.editDialog.detailLabel') }}</span>
            <div class="product-edit-drawer__detail-row">
              <img
                v-for="(src, index) in detailPreviewSrcs"
                :key="`${src}-${index}`"
                class="product-edit-drawer__thumb"
                :src="src"
                :alt="t('products.editDialog.detailLabel')"
              />
            </div>
          </div>
          <p class="product-edit-drawer__images-hint">{{ t('products.editDialog.imagesReadOnlyHint') }}</p>
        </div>
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="product-edit-drawer__footer">
        <NButton :disabled="submitting" @click="close">{{ t('products.editDialog.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('products.editDialog.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.product-edit-drawer__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.product-edit-drawer__images {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  width: 100%;
}

.product-edit-drawer__cover-block,
.product-edit-drawer__detail-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.product-edit-drawer__images-caption {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.product-edit-drawer__detail-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.product-edit-drawer__thumb {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-sm, 4px);
  object-fit: cover;
  background: var(--color-surface-muted, #f0f0f0);
}

.product-edit-drawer__thumb--cover {
  width: 72px;
  height: 72px;
}

.product-edit-drawer__thumb--placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--color-border, #d0d0d0);
  font-family: var(--font-body);
  font-size: var(--font-size-xs, 11px);
  color: var(--color-text-muted);
  text-align: center;
  padding: var(--space-1);
}

.product-edit-drawer__images-hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs, 11px);
  color: var(--color-text-muted);
}
</style>
