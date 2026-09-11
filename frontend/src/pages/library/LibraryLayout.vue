<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NTab, NTabs } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activeTab = computed({
  get() {
    if (route.path.startsWith('/library/products')) return 'products'
    if (route.path.startsWith('/library/templates')) return 'templates'
    return 'customers'
  },
  set(val: string) {
    void router.push(`/library/${val}`)
  },
})
</script>

<template>
  <div class="library-layout">
    <PageHeader :title="t('library.title')" :description="t('library.subtitle')" />

    <div class="library-layout__tabs">
      <NTabs v-model:value="activeTab" type="line" animated>
        <NTab name="customers">{{ t('library.customersTab') }}</NTab>
        <NTab name="products">{{ t('library.productsTab') }}</NTab>
        <NTab name="templates">{{ t('library.templatesTab') }}</NTab>
      </NTabs>
    </div>

    <div class="library-layout__content">
      <RouterView />
    </div>
  </div>
</template>

<style scoped>
.library-layout {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.library-layout__tabs {
  margin-bottom: var(--space-2);
}

.library-layout__content {
  min-height: 400px;
}
</style>
