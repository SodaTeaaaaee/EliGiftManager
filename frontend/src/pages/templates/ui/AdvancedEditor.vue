<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NCard, NDataTable } from 'naive-ui'
import type { DynamicTemplateRules } from './types'

const props = defineProps<{ templateConfig: DynamicTemplateRules }>()

const jsonText = ref(JSON.stringify(props.templateConfig, null, 2))
const jsonError = ref('')
const sampleCsv = ref<string[][]>([
  ['platform_uid', 'gift_level', 'nickname', 'recipient_name', 'phone', 'address', 'extra_field'],
  ['12345', '总督', '小明', '张三', '13800138000', '上海市浦东新区', 'some extra'],
  ['67890', '提督', '小红', '李四', '13900139000', '北京市朝阳区', 'more extra'],
])

// JSON text → templateConfig
watch(jsonText, (val) => {
  try {
    const parsed = JSON.parse(val)
    Object.assign(props.templateConfig, parsed)
    jsonError.value = ''
  } catch (e) {
    jsonError.value = String(e)
  }
})

// templateConfig → JSON text (external changes from BasicMapper)
watch(
  () => props.templateConfig,
  () => {
    jsonText.value = JSON.stringify(props.templateConfig, null, 2)
  },
  { deep: true },
)

// JS approximation of Go ParseRowDynamically — for sandbox preview only.
// Known differences: columnIndex priority reversed vs Go, header normalisation is simpler (no BOM strip).
function parseRowDynamicallyJS(record: string[], headers: string[], rules: DynamicTemplateRules) {
  const coreData: Record<string, string> = {}
  const extraData: Record<string, string> = {}
  const errors: string[] = []
  const usedIndices = new Set<number>()

  for (const [key, mapping] of Object.entries(rules.mapping)) {
    let idx: number
    if (mapping.columnIndex !== undefined && mapping.columnIndex >= 0) {
      idx = mapping.columnIndex
    } else if (rules.hasHeader && mapping.sourceColumn) {
      idx = headers.findIndex(
        (h) =>
          h.toLowerCase().replace(/[ _-]/g, '') ===
          mapping.sourceColumn.toLowerCase().replace(/[ _-]/g, ''),
      )
    } else {
      idx = -1
    }

    const value = idx >= 0 && idx < record.length ? record[idx].trim() : ''
    if (!value && mapping.required) {
      errors.push(`${key} 是必填字段`)
      continue
    }
    if (!value && mapping.defaultValue) {
      coreData[key] = mapping.defaultValue
      continue
    }
    if (idx >= 0) usedIndices.add(idx)
    coreData[key] = value
  }

  if (rules.extraData?.strategy === 'catch_all') {
    for (let i = 0; i < record.length; i++) {
      if (!usedIndices.has(i) && record[i].trim()) {
        extraData[headers[i] || `column_${i}`] = record[i].trim()
      }
    }
  }

  return { coreData, extraData, errors }
}

interface PreviewRow {
  row: number
  coreData: Record<string, string>
  extraData: Record<string, string>
  errors: string[]
}

const previewResult = computed<PreviewRow[]>(() => {
  if (!sampleCsv.value.length) return []
  const headers = sampleCsv.value[0] || []
  return sampleCsv.value.slice(1).map((row, i) => ({
    row: i + 1,
    ...parseRowDynamicallyJS(row, headers, props.templateConfig),
  }))
})

const previewColumns = computed(() => {
  const keys = Object.keys(props.templateConfig.mapping)
  return keys.map((k) => ({ title: k, key: k, width: 100 }))
})
</script>

<template>
  <div class="flex gap-4" style="min-height: 480px">
    <!-- Left: JSON editor -->
    <div class="flex-1" style="flex-basis: 60%">
      <p class="text-sm font-medium mb-2">JSON 规则编辑</p>
      <textarea
        v-model="jsonText"
        class="w-full font-mono text-sm p-3 border rounded resize-none"
        style="height: 440px"
        spellcheck="false"
      />
      <NAlert v-if="jsonError" type="error" class="mt-2"> JSON 解析错误：{{ jsonError }} </NAlert>
    </div>

    <!-- Right: Sandbox -->
    <div style="flex-basis: 40%; min-width: 320px" class="space-y-3">
      <NCard title="示例 CSV（前 3 行）" size="small">
        <table class="w-full text-xs">
          <thead>
            <tr>
              <th
                v-for="(h, i) in sampleCsv[0] || []"
                :key="i"
                class="text-left px-1 py-0.5 border-b"
              >
                {{ h }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, ri) in sampleCsv.slice(1)" :key="ri">
              <td
                v-for="(cell, ci) in row"
                :key="ci"
                class="px-1 py-0.5 border-b border-gray-100 dark:border-gray-700"
              >
                {{ cell }}
              </td>
            </tr>
          </tbody>
        </table>
      </NCard>

      <NCard title="实时预览" size="small">
        <div
          v-for="(r, i) in previewResult"
          :key="i"
          class="mb-3 pb-3 border-b border-gray-100 dark:border-gray-700 last:border-0 last:mb-0 last:pb-0"
        >
          <p class="text-xs text-gray-500 mb-1">第 {{ r.row }} 行</p>
          <NAlert v-if="r.errors.length" type="error" class="mb-1">
            <div v-for="(err, ei) in r.errors" :key="ei">{{ err }}</div>
          </NAlert>
          <div class="text-xs">
            <p class="font-medium mb-0.5">Core Data:</p>
            <div v-for="(val, key) in r.coreData" :key="key" class="flex gap-2 ml-2">
              <span class="text-gray-500">{{ key }}:</span>
              <span>{{ val || '-' }}</span>
            </div>
            <p v-if="Object.keys(r.extraData).length" class="font-medium mt-1 mb-0.5">
              Extra Data:
            </p>
            <div v-for="(val, key) in r.extraData" :key="key" class="flex gap-2 ml-2">
              <span class="text-gray-500">{{ key }}:</span>
              <span>{{ val }}</span>
            </div>
          </div>
        </div>
        <p v-if="!previewResult.length" class="text-xs text-gray-400">暂无预览数据</p>
      </NCard>
    </div>
  </div>
</template>
