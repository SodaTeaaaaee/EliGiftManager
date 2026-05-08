<script setup lang="ts">
import { AlertCircleOutline, CloudDownloadOutline, CloudUploadOutline } from '@vicons/ionicons5'
import { onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NIcon,
  NRadioButton,
  NRadioGroup,
  useDialog,
  useMessage,
} from 'naive-ui'
import {
  backupDatabase,
  bootstrapApp,
  createFakeAddresses,
  deleteFakeAddresses,
  getDashboard,
  isWailsRuntimeAvailable,
  pingDatabase,
  restoreDatabase,
  WAILS_PREVIEW_MESSAGE,
  type DashboardPayload,
} from '@/shared/lib/wails/app'
import { themePreferenceOptions, useThemeStore, type ThemePreference } from '@/shared/model/theme'
import { useScrollMode } from '@/shared/model/settings'

const message = useMessage()
const dialog = useDialog()
const themeStore = useThemeStore()
const scrollMode = useScrollMode()

function setScrollMode(v: string | number | boolean) {
  scrollMode.value = v === 'scroll'
}
const dashboard = ref<DashboardPayload | null>(null)
const dbStatus = ref('等待检测')
const errorMessage = ref('')
const isCreatingFakeAddresses = ref(false)
const isDeletingFakeAddresses = ref(false)

function handleThemeChange(value: string | number | boolean) {
  themeStore.setPreference(value as ThemePreference)
}
async function loadSettings() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }
  try {
    await bootstrapApp()
    dashboard.value = await getDashboard()
    dbStatus.value = await pingDatabase()
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载设置失败。'
  }
}
async function handleBackup() {
  try {
    const path = await backupDatabase()
    message.success(`备份完成：${path}`)
  } catch (error) {
    message.error(String(error))
  }
}
function handleRestore() {
  dialog.warning({
    title: '危险操作确认',
    content: '恢复数据库会覆盖当前数据，系统会先自动保存一份灾备副本。是否继续？',
    positiveText: '确认恢复',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await restoreDatabase()
        message.success('数据库已恢复，请重启应用以刷新所有连接状态。')
        await loadSettings()
      } catch (error) {
        message.error(String(error))
      }
    },
  })
}

async function handleCreateFakeAddresses() {
  try {
    const result = await createFakeAddresses()
    message.success(
      `已创建 ${result.created} 条测试地址，共 ${result.totalMembers} 个会员，` +
        `跳过 ${result.skippedHasAddress} 个已有地址会员`,
    )
    await loadSettings()
  } catch (error) {
    message.error(String(error))
  }
}

async function handleDeleteFakeAddresses() {
  try {
    const result = await deleteFakeAddresses()
    const parts: string[] = []
    if (result.deletedAddresses > 0) parts.push(`已删除 ${result.deletedAddresses} 条测试地址`)
    if (result.clearedDispatchRecords > 0)
      parts.push(`已清空 ${result.clearedDispatchRecords} 条发货记录`)
    if (result.updatedWaves > 0) parts.push(`已回退 ${result.updatedWaves} 个波次状态`)
    if (parts.length === 0) {
      message.info('没有找到测试地址，无需清理。')
    } else {
      message.success(parts.join('，') + '。')
    }
    await loadSettings()
  } catch (error) {
    message.error(String(error))
  }
}

onMounted(loadSettings)
</script>
<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Settings</p>
      <h1 class="app-title mt-2">系统设置</h1>
      <p class="app-copy mt-2">管理界面主题、SQLite 数据文件和运行时信息。</p>
    </header>
    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>
    <div class="grid gap-4 xl:grid-cols-2">
      <NCard title="界面主题" size="medium">
        <p class="app-copy mb-3">选择浅色、深色，或跟随操作系统。</p>
        <NRadioGroup :value="themeStore.preference" @update:value="handleThemeChange">
          <NRadioButton
            v-for="option in themePreferenceOptions"
            :key="option.value"
            :value="option.value"
            >{{ option.label }}</NRadioButton
          >
        </NRadioGroup>
      </NCard>
      <NCard title="表格模式" size="medium">
        <p class="app-copy mb-3">自适应分页或传统滚动。</p>
        <NRadioGroup :value="scrollMode ? 'scroll' : 'paginated'" @update:value="setScrollMode">
          <NRadioButton value="paginated">自适应分页</NRadioButton>
          <NRadioButton value="scroll">滚动模式</NRadioButton>
        </NRadioGroup>
      </NCard>
      <NCard title="数据库状态" size="medium">
        <p class="app-copy">{{ dbStatus }}</p>
        <p class="app-copy mt-3 break-all">{{ dashboard?.databasePath ?? '等待连接' }}</p>
      </NCard>
      <NCard title="数据安全" size="medium" class="xl:col-span-2">
        <div class="grid gap-3 md:grid-cols-2">
          <NButton size="large" type="primary" @click="handleBackup"
            ><template #icon>
              <NIcon>
                <CloudDownloadOutline />
              </NIcon> </template
            >导出数据备份 (.db)</NButton
          >
          <NButton size="large" type="error" ghost @click="handleRestore"
            ><template #icon>
              <NIcon>
                <CloudUploadOutline />
              </NIcon> </template
            >从备份恢复数据</NButton
          >
        </div>
        <NAlert class="mt-4" type="warning" :show-icon="false"
          ><span class="inline-flex items-center gap-2">
            <NIcon> <AlertCircleOutline /> </NIcon>恢复前会自动生成带时间戳的防灾副本。
          </span></NAlert
        >
      </NCard>
      <NCard title="测试地址工具" size="medium" class="xl:col-span-2">
        <p class="app-copy mb-3">
          为缺少有效地址的会员批量生成虚拟地址，用于测试完整发货流程。删除操作会清理所有测试地址及其关联记录。
        </p>
        <div class="grid gap-3 md:grid-cols-2">
          <NButton
            size="large"
            type="primary"
            :loading="isCreatingFakeAddresses"
            @click="
              dialog.warning({
                title: '确认生成测试地址',
                content:
                  '将为当前没有有效地址的所有会员各添加一条默认测试地址（标记为 __ELIGIFT_TEST_ADDRESS__）。已有地址的会员不受影响。',
                positiveText: '确认生成',
                negativeText: '取消',
                onPositiveClick: async () => {
                  isCreatingFakeAddresses = true
                  try {
                    await handleCreateFakeAddresses()
                  } finally {
                    isCreatingFakeAddresses = false
                  }
                },
              })
            "
          >
            一键生成测试地址
          </NButton>
          <NButton
            size="large"
            type="error"
            ghost
            :loading="isDeletingFakeAddresses"
            @click="
              dialog.warning({
                title: '确认删除测试地址',
                content:
                  '将删除所有系统生成的测试地址（标记为 __ELIGIFT_TEST_ADDRESS__），并清空引用这些地址的发货记录绑定。已导出的波次状态将回退为待补全。此操作不可撤销。',
                positiveText: '确认删除',
                negativeText: '取消',
                onPositiveClick: async () => {
                  isDeletingFakeAddresses = true
                  try {
                    await handleDeleteFakeAddresses()
                  } finally {
                    isDeletingFakeAddresses = false
                  }
                },
              })
            "
          >
            一键删除测试地址
          </NButton>
        </div>
      </NCard>
    </div>
  </section>
</template>
