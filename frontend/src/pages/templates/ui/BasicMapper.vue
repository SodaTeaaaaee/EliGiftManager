<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NButton, NSelect, NSwitch, NTag } from 'naive-ui'
import type { DynamicTemplateRules } from './types'

const props = defineProps<{ templateConfig: DynamicTemplateRules; csvHeaders: string[] }>()
defineEmits<{ upload: [] }>()

const headerLoaded = computed(() => props.csvHeaders.length > 0)
const csvError = ref('')

function getOptions(currentFieldKey: string) {
  if (!props.templateConfig.hasHeader) {
    const count = props.csvHeaders.length
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
  const m = props.templateConfig.mapping[fieldKey]
  if (!m) return
  if (props.templateConfig.hasHeader) {
    m.sourceColumn = v
    m.columnIndex = undefined
  } else {
    m.columnIndex = Number(v)
    m.sourceColumn = ''
  }
}

const coreFields = [
  { key: 'platform_uid', label: '平台 UID', required: true },
  { key: 'gift_level', label: '等级', required: true },
  { key: 'nickname', label: '昵称', required: false },
]

const shippingFields = [
  { key: 'recipient_name', label: '收件人', required: false },
  { key: 'phone', label: '电话', required: false },
  { key: 'address', label: '地址', required: false },
]
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

    <div v-if="headerLoaded" class="flex items-center gap-2">
      <NSwitch v-model:value="templateConfig.hasHeader" />
      <span class="text-sm">第一行是表头</span>
      <NTag v-if="!templateConfig.hasHeader" size="tiny" type="warning">
        使用列索引映射
      </NTag>
      <NTag v-else size="tiny" type="info">
        使用表头名映射
      </NTag>
    </div>

    <div v-if="headerLoaded">
      <p class="text-sm font-medium mb-2">Step 2：映射字段</p>

      <div class="mb-3">
        <p class="text-xs text-gray-500 mb-1">核心身份</p>
        <div class="space-y-2">
          <div v-for="field in coreFields" :key="field.key" class="flex items-center gap-2">
            <label class="text-sm w-20 shrink-0">
              {{ field.label }}
              <span
                v-if="templateConfig.mapping[field.key]?.required"
                class="text-red-500"
              >*</span>
            </label>
            <NSelect
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

      <div>
        <p class="text-xs text-gray-500 mb-1">发货信息（可选）</p>
        <div class="space-y-2">
          <div v-for="field in shippingFields" :key="field.key" class="flex items-center gap-2">
            <label class="text-sm w-20 shrink-0">{{ field.label }}</label>
            <NSelect
              :value="selectedValue(field.key)"
              :options="getOptions(field.key)"
              :placeholder="field.label"
              class="flex-1"
              clearable
              @update:value="(v: string) => setSelectedValue(field.key, v)"
            />
          </div>
        </div>
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
