<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NCard } from 'naive-ui'
import type { DynamicTemplateRules } from './types'

const props = defineProps<{ templateConfig: DynamicTemplateRules; sampleRows: string[][] }>()

const jsonText = ref(JSON.stringify(props.templateConfig, null, 2))
const jsonError = ref('')

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
// Kept in sync with Go normalizeDynamicKey() in internal/service/dynamic_parser.go.
// When changing normalize rules, update this implementation AND the Go comment too.
function parseRowDynamicallyJS(record: string[], headers: string[], rules: DynamicTemplateRules) {
  const coreData: Record<string, string> = {}
  const extraData: Record<string, string> = {}
  const errors: string[] = []
  const usedIndices = new Set<number>()

  function normalizeKey(key: string): string {
    return key.replace(/^﻿/, '').toLowerCase().replace(/[ _-]/g, '')
  }

  for (const [key, mapping] of Object.entries(rules.mapping)) {
    let idx: number
    if (rules.hasHeader && mapping.sourceColumn) {
      idx = headers.findIndex((h) => normalizeKey(h) === normalizeKey(mapping.sourceColumn!))
    } else {
      idx = mapping.columnIndex ?? -1
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
  if (!props.sampleRows.length) return []
  const headers = props.sampleRows[0] || []
  return props.sampleRows.slice(1).map((row: string[], i: number) => ({
    row: i + 1,
    ...parseRowDynamicallyJS(row, headers, props.templateConfig),
  }))
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
                v-for="(h, i) in sampleRows[0] || []"
                :key="i"
                class="text-left px-1 py-0.5 border-b"
              >
                {{ h }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, ri) in sampleRows.slice(1)" :key="ri">
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
        <p class="text-xs text-gray-400 mb-2">预览解析逻辑已与导入引擎对齐</p>
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
