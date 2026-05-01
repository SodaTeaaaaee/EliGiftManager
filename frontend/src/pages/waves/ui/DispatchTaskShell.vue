<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { NButton, NStep, NSteps, NTag } from 'naive-ui'
import { listWaves, type WaveItem } from '@/shared/lib/wails/app'

const router = useRouter()
const route = useRoute()
const wave = ref<WaveItem | null>(null)

const selectedWaveId = computed(() => {
  const id = Number(route.params.waveId)
  return id > 0 ? id : null
})

const currentStep = computed(() => {
  const name = route.name
  if (name === 'waves-step-import') return 1
  if (name === 'waves-step-tags') return 2
  if (name === 'waves-step-preview') return 3
  if (name === 'waves-step-export') return 4
  return 1
})

const isEditing = computed(() => !!selectedWaveId.value)

function statusLabel(s: string) {
  return ({ draft: '草稿', allocating: '分配中', pending_address: '待补全', exported: '已导出' } as Record<string, string>)[s] ?? s
}
function statusTagType(s: string) {
  return s === 'exported' ? 'success' : s === 'pending_address' ? 'warning' : s === 'allocating' ? 'info' : 'default'
}

async function loadWave() {
  if (!selectedWaveId.value) { wave.value = null; return }
  try {
    const waves = await listWaves()
    wave.value = waves.find(w => w.id === selectedWaveId.value) ?? null
  } catch { wave.value = null }
}

function closeWave() { router.push({ name: 'waves-welcome' }) }

onMounted(loadWave)

watch(() => route.params.waveId, () => loadWave())
</script>
<template>
  <div class="h-full flex flex-col">
    <!-- 编辑模式顶栏 -->
    <div v-if="isEditing && wave" class="flex flex-wrap items-center gap-3 px-1 py-2 shrink-0">
      <span class="font-semibold text-lg min-w-0 break-all">{{ wave.waveNo }} · {{ wave.name }}</span>
      <NTag :type="statusTagType(wave.status)" size="small" round>{{ statusLabel(wave.status) }}</NTag>
      <div class="ml-auto">
        <NButton size="small" secondary @click="closeWave">关闭任务</NButton>
      </div>
    </div>

    <!-- 步骤条（仅编辑模式） -->
    <NSteps v-if="isEditing" :current="currentStep" status="process" class="shrink-0 mb-2">
      <NStep title="导入数据" />
      <NStep title="Tag 管理与分配" />
      <NStep title="导出预览与编辑" />
      <NStep title="异常检查与导出" />
    </NSteps>

    <!-- 唯一 RouterView -->
    <RouterView class="flex-1 min-h-0" />
  </div>
</template>
