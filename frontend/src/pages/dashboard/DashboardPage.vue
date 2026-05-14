<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NDataTable, NAlert, type DataTableColumns } from 'naive-ui'
import { listWaves } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'

const router = useRouter()
const waves = ref<dto.WaveDTO[]>([])
const loading = ref(false)
const error = ref('')

const columns: DataTableColumns<dto.WaveDTO> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '波次号', key: 'waveNo', width: 180 },
  { title: '名称', key: 'name' },
  {
    title: '阶段',
    key: 'lifecycleStage',
    width: 120,
    render(row) {
      return h('span', row.lifecycleStage || '-')
    },
  },
]

function handleRowClick(row: dto.WaveDTO) {
  router.push(`/waves/${row.id}`)
}

async function loadWaves() {
  loading.value = true
  error.value = ''
  try {
    waves.value = await listWaves()
  } catch (e: any) {
    error.value = e?.message ?? String(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadWaves)
</script>

<template>
  <div class="dashboard-page p-4">
    <h1 class="text-xl font-medium mb-4">仪表盘</h1>

    <n-alert v-if="error" type="error" :title="error" class="mb-4" />

    <n-card title="活跃波次">
      <n-data-table
        :columns="columns"
        :data="waves"
        :loading="loading"
        :pagination="false"
        size="small"
        :row-props="(row: dto.WaveDTO) => ({
          style: 'cursor: pointer',
          onClick: () => handleRowClick(row),
        })"
      />
    </n-card>
  </div>
</template>
