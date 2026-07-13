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
import type { ProductMaster } from '@/entities/product'

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
</style>
