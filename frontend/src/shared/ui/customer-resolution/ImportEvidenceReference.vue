<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { resolveImportEvidenceReference } from '@/shared/lib/customer-resolution/importEvidenceReference'

const props = defineProps<{ importRunId?: number; evidenceDisabled?: boolean }>()
const { t } = useI18n({ useScope: 'global' })
const reference = computed(() => resolveImportEvidenceReference(props))
</script>

<template>
  <p class="import-evidence-reference">
    {{ reference.kind === 'run'
      ? t('settings.importEvidence.referenceRun', { id: reference.importRunId })
      : t('settings.importEvidence.referenceDisabled') }}
  </p>
</template>

<style scoped>
.import-evidence-reference {
  margin: 0;
  color: var(--color-text-secondary);
  font-family: var(--font-mono, monospace);
  font-size: var(--font-size-xs);
}
</style>
