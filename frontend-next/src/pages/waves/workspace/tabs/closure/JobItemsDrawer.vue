<script setup lang="ts">
/**
 * JobItemsDrawer — item-level detail for one `ChannelSyncJob` row
 * (`WaveClosureTab.vue`'s jobs table "view items" action). `DataGrid.vue`
 * (the house `NDataTable` wrapper) has no row-expand affordance, so item
 * detail is a side drawer instead of an inline expanded row — reuses
 * `DetailDrawer` (the house "inspect one thing without leaving the grid"
 * primitive) rather than inventing a new pattern.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { DetailDrawer } from '@/shared/ui/drawer'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  show: boolean
  job: dto.ChannelSyncJobDTO | null
}>()

const emit = defineEmits<{ 'update:show': [boolean] }>()

const { t } = useI18n({ useScope: 'global' })

const items = computed(() => props.job?.items ?? [])

const columns = computed(() =>
  createColumns<dto.ChannelSyncItemDTO>([
    {
      type: 'text',
      key: 'externalDocumentNo',
      title: t('waveWorkspace.closure.jobs.itemColumns.externalDocumentNo'),
      minWidth: 140,
    },
    {
      type: 'text',
      key: 'externalLineNo',
      title: t('waveWorkspace.closure.jobs.itemColumns.externalLineNo'),
      width: 110,
    },
    {
      type: 'text',
      key: 'carrierCode',
      title: t('waveWorkspace.closure.jobs.itemColumns.carrierCode'),
      width: 110,
    },
    {
      type: 'text',
      key: 'trackingNo',
      title: t('waveWorkspace.closure.jobs.itemColumns.trackingNo'),
      minWidth: 140,
    },
    {
      type: 'status',
      key: 'status',
      title: t('waveWorkspace.closure.jobs.itemColumns.status'),
      dimension: 'channelSyncItemStatus',
      width: 100,
    },
    {
      type: 'text',
      key: 'errorMessage',
      title: t('waveWorkspace.closure.jobs.itemColumns.error'),
      minWidth: 160,
    },
  ]),
)
</script>

<template>
  <DetailDrawer
    :show="show"
    :title="job ? t('waveWorkspace.closure.jobs.columns.id') + ` #${job.id}` : ''"
    size="lg"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <DataGrid
      :columns="columns"
      :rows="items"
      row-key="id"
      pagination="client"
      :empty="{ title: t('waveWorkspace.closure.jobs.empty') }"
    />
  </DetailDrawer>
</template>
