<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NButton, NInput, NInputNumber, NSelect, NSwitch, NTag } from 'naive-ui'
import type { DynamicTemplateRules } from './types'

export interface FieldDef {
  key: string
  label: string
  required?: boolean
}

export interface FieldGroup {
  label: string
  fields: FieldDef[]
}

const props = defineProps<{
  templateConfig: DynamicTemplateRules
  csvHeaders: string[]
  fieldGroups: FieldGroup[]
}>()
defineEmits<{ upload: [] }>()

const headerLoaded = computed(() => props.csvHeaders.length > 0)
const csvError = ref('')
const customIndexField = ref('')

const maxExistingColumnIndex = computed(() => {
  let max = -1
  for (const m of Object.values(props.templateConfig.mapping)) {
    if (m.columnIndex !== undefined && m.columnIndex > max) {
      max = m.columnIndex
    }
  }
  return max
})

const customIndexFieldOptions = computed(() =>
  Object.keys(props.templateConfig.mapping).map((k) => ({ label: k, value: k })),
)

function getOptions(currentFieldKey: string) {
  if (!props.templateConfig.hasHeader) {
    const count = Math.max(props.csvHeaders.length, maxExistingColumnIndex.value + 5, 20)
    return Array.from({ length: count }, (_, i) => {
      const preview = props.csvHeaders[i] || ''
      return {
        label: `列 ${i}` + (preview ? `（${preview}）` : ''),
        value: String(i),
      }
    })
  }
  const usedHeaders = new Set<string>()
  for (const [key, mapping] of Object.entries(props.templateConfig.mapping)) {
    if (key !== currentFieldKey && mapping.sourceColumn) {
      usedHeaders.add(mapping.sourceColumn)
    }
  }
  return props.csvHeaders.map((h) => ({
    label: h,
    value: h,
    disabled: usedHeaders.has(h),
  }))
}

function selectedValue(fieldKey: string): string | undefined {
  const m = props.templateConfig.mapping[fieldKey]
  if (!m) return undefined
  return props.templateConfig.hasHeader
    ? m.sourceColumn || undefined
    : m.columnIndex !== undefined ? String(m.columnIndex) : undefined
}

function setSelectedValue(fieldKey: string, v: string) {
  let m = props.templateConfig.mapping[fieldKey]
  if (!m) {
    m = { columnIndex: undefined, sourceColumn: '' }
    props.templateConfig.mapping[fieldKey] = m
  }
  if (props.templateConfig.hasHeader) {
    m.sourceColumn = v
    m.columnIndex = undefined
  } else {
    m.columnIndex = Number(v)
    m.sourceColumn = ''
  }
}
</script>

<template>
  <div class="border rounded-lg p-4 space-y-4">
    <div>
      <p class="text-sm font-medium mb-2">Step 1：上传示例 CSV</p>
      <NButton @click="$emit('upload')">上传示例 CSV</NButton>
      <NAlert v-if="headerLoaded" type="info" class="mt-2">
        <template v-if="templateConfig.hasHeader">
          已检测到 {{ props.csvHeaders.length }} 个列：{{ props.csvHeaders.join(', ') }}
        </template>
        <template v-else>
          已检测到 {{ props.csvHeaders.length }} 列（第一行作为数据行，不视为表头）
        </template>
      </NAlert>
      <NAlert v-if="csvError" type="error" class="mt-2">
        {{ csvError }}
      </NAlert>
    </div>

    <div class="flex items-center gap-2">
      <NSwitch v-model:value="templateConfig.hasHeader" />
      <span class="text-sm">第一行是表头</span>
      <NTag v-if="!templateConfig.hasHeader" size="tiny" type="warning">
        使用列索引映射
      </NTag>
      <NTag v-else size="tiny" type="info">
        使用表头名映射
      </NTag>
    </div>

    <div>
      <p class="text-sm font-medium mb-2">Step 2：映射字段</p>

      <template v-for="(group, gi) in fieldGroups" :key="gi">
        <div :class="gi > 0 ? 'mt-3' : ''">
          <p class="text-xs text-gray-500 mb-1">{{ group.label }}</p>
          <div class="space-y-2">
            <div v-for="field in group.fields" :key="field.key" class="flex items-center gap-2">
              <label class="text-sm w-20 shrink-0">
                {{ field.label }}
                <span
                  v-if="templateConfig.mapping[field.key]?.required"
                  class="text-red-500"
                >*</span>
              </label>
              <NInput
                v-if="templateConfig.hasHeader && !headerLoaded"
                :value="selectedValue(field.key)"
                :placeholder="field.label"
                class="flex-1"
                clearable
                @update:value="(v: string) => setSelectedValue(field.key, v)"
              />
              <NSelect
                v-else
                :value="selectedValue(field.key)"
                :options="getOptions(field.key)"
                :placeholder="field.label"
                :required="templateConfig.mapping[field.key]?.required"
                class="flex-1"
                clearable
                @update:value="(v: string) => setSelectedValue(field.key, v)"
              />
            </div>
          </div>
        </div>
      </template>

      <div v-if="!templateConfig.hasHeader" class="mt-2 pt-2 border-t border-gray-100 dark:border-gray-700 flex items-center gap-2">
        <span class="text-xs text-gray-500">快速指定列号：</span>
        <NSelect
          v-model:value="customIndexField"
          :options="customIndexFieldOptions"
          size="tiny"
          style="width: 120px"
          placeholder="字段"
        />
        <span class="text-xs text-gray-400">→ 列</span>
        <NInputNumber
          :min="0"
          size="tiny"
          style="width: 100px"
          placeholder="列号"
          @update:value="(v: number | null) => {
            if (v != null && customIndexField) setSelectedValue(customIndexField, String(v))
          }"
        />
      </div>

      <NAlert type="info" class="mt-3">
        <template v-if="templateConfig.hasHeader">
          未映射的列将自动保存至 Extra Data（catch_all 模式）
        </template>
        <template v-else>
          未使用的列将自动保存至 Extra Data（catch_all 模式）
        </template>
      </NAlert>
    </div>
  </div>
</template>
