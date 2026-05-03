<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { NButton, NInput, NSelect, NSwitch, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import { createTemplate } from '@/shared/lib/wails/app'
import BasicMapper from './BasicMapper.vue'
import AdvancedEditor from './AdvancedEditor.vue'
import type { DynamicTemplateRules } from './types'

const message = useMessage()
const router = useRouter()

const platformOptions = [
  { label: 'BILIBILI', value: 'BILIBILI' },
  { label: 'DOUYIN', value: 'DOUYIN' },
  { label: 'KUAISHOU', value: 'KUAISHOU' },
  { label: 'XIAOHONGSHU', value: 'XIAOHONGSHU' },
  { label: 'WEIBO', value: 'WEIBO' },
  { label: 'ACFUN', value: 'ACFUN' },
  { label: 'YOUTUBE', value: 'YOUTUBE' },
  { label: 'TWITCH', value: 'TWITCH' },
  { label: 'OTHER', value: 'OTHER' },
]

const templateConfig = reactive<DynamicTemplateRules>({
  format: 'csv',
  hasHeader: true,
  mapping: {
    platform_uid: { sourceColumn: '', required: true },
    gift_level: { sourceColumn: '', required: true },
    nickname: { sourceColumn: '', required: false },
    recipient_name: { sourceColumn: '', required: false },
    phone: { sourceColumn: '', required: false },
    address: { sourceColumn: '', required: false },
  },
  extraData: { strategy: 'catch_all' },
})

const isAdvanced = ref(false)
const templateName = ref('')
const templatePlatform = ref('')

function onAdvancedChange(val: boolean) {
  if (!val && templateConfig.extraData.strategy !== 'catch_all') {
    message.warning('切换回基础模式将丢失高级 ExtraData 设置')
  }
}

async function handleSave() {
  if (!templateName.value.trim()) {
    message.warning('请输入模板名称')
    return
  }
  if (!templatePlatform.value.trim()) {
    message.warning('请选择平台')
    return
  }
  try {
    await createTemplate(
      templatePlatform.value,
      'import_dispatch_record',
      templateName.value,
      JSON.stringify(templateConfig),
    )
    message.success('模板已保存')
    router.push({ name: 'templates' })
  } catch (e) {
    message.error(String(e))
  }
}

function handleCancel() {
  router.push({ name: 'templates' })
}
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Templates</p>
      <h1 class="app-title mt-2">自定义模板构建器</h1>
      <p class="app-copy mt-2">
        通过 CSV 示例文件快速建立字段映射，或切换到高级模式直接编辑 JSON 规则。
      </p>
    </header>

    <div class="flex items-center gap-3">
      <NInput v-model:value="templateName" placeholder="模板名称" style="max-width: 240px" />
      <NSelect
        v-model:value="templatePlatform"
        :options="platformOptions"
        placeholder="选择平台"
        style="max-width: 180px"
      />
    </div>

    <div class="flex items-center gap-2">
      <NSwitch v-model:value="isAdvanced" @update:value="onAdvancedChange" />
      <span class="text-sm" :class="isAdvanced ? 'text-[var(--primary)]' : ''">
        {{ isAdvanced ? '高级模式' : '基础模式' }}
      </span>
    </div>

    <BasicMapper v-if="!isAdvanced" :template-config="templateConfig" />
    <AdvancedEditor v-else :template-config="templateConfig" />

    <div class="flex gap-2 pt-2">
      <NButton type="primary" @click="handleSave">保存模板</NButton>
      <NButton secondary @click="handleCancel">取消</NButton>
    </div>
  </section>
</template>
