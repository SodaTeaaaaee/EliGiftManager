<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NButton, NSelect } from 'naive-ui'
import type { DynamicTemplateRules } from './types'

const props = defineProps<{ templateConfig: DynamicTemplateRules; csvHeaders: string[] }>()
defineEmits<{ upload: [] }>()

const headerLoaded = computed(() => props.csvHeaders.length > 0)
const csvError = ref('')

function getOptions(currentFieldKey: string) {
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
        已检测到 {{ props.csvHeaders.length }} 个列：{{ props.csvHeaders.join(', ') }}
      </NAlert>
      <NAlert v-if="csvError" type="error" class="mt-2">
        {{ csvError }}
      </NAlert>
    </div>

    <div v-if="headerLoaded">
      <p class="text-sm font-medium mb-2">Step 2：映射字段</p>

      <div class="mb-3">
        <p class="text-xs text-gray-500 mb-1">核心身份</p>
        <div class="space-y-2">
          <div v-for="field in coreFields" :key="field.key" class="flex items-center gap-2">
            <label class="text-sm w-20 shrink-0">
              {{ field.label }}
              <span v-if="templateConfig.mapping[field.key]?.required" class="text-red-500">*</span>
            </label>
            <NSelect
              :value="templateConfig.mapping[field.key]?.sourceColumn || ''"
              :options="getOptions(field.key)"
              :placeholder="field.label"
              :required="templateConfig.mapping[field.key]?.required"
              class="flex-1"
              clearable
              @update:value="
                (v: string) => {
                  if (templateConfig.mapping[field.key])
                    templateConfig.mapping[field.key].sourceColumn = v
                }
              "
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
              :value="templateConfig.mapping[field.key]?.sourceColumn || ''"
              :options="getOptions(field.key)"
              :placeholder="field.label"
              class="flex-1"
              clearable
              @update:value="
                (v: string) => {
                  if (templateConfig.mapping[field.key])
                    templateConfig.mapping[field.key].sourceColumn = v
                }
              "
            />
          </div>
        </div>
      </div>

      <NAlert type="info" class="mt-3">
        未映射的列将自动保存至 Extra Data（catch_all 模式）
      </NAlert>
    </div>
  </div>
</template>
