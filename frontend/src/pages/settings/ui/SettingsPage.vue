<script setup lang="ts">
import { AlertCircleOutline, CloudDownloadOutline, CloudUploadOutline } from '@vicons/ionicons5'
import { onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NIcon, NRadioButton, NRadioGroup, useDialog, useMessage } from 'naive-ui'
import { backupDatabase, bootstrapApp, getDashboard, isWailsRuntimeAvailable, pingDatabase, restoreDatabase, WAILS_PREVIEW_MESSAGE, type DashboardPayload } from '@/shared/lib/wails/app'
import { themePreferenceOptions, useThemeStore, type ThemePreference } from '@/shared/model/theme'

const message = useMessage()
const dialog = useDialog()
const themeStore = useThemeStore()
const dashboard = ref<DashboardPayload | null>(null)
const dbStatus = ref('等待检测')
const errorMessage = ref('')

function handleThemeChange(value: string | number | boolean) {
  themeStore.setPreference(value as ThemePreference)
}
async function loadSettings() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return }
  try { await bootstrapApp(); dashboard.value = await getDashboard(); dbStatus.value = await pingDatabase() } catch (error) { console.error(error); errorMessage.value = '加载设置失败。' }
}
async function handleBackup() { try { const path = await backupDatabase(); message.success(`备份完成：${path}`) } catch (error) { message.error(String(error)) } }
function handleRestore() { dialog.warning({ title: '危险操作确认', content: '恢复数据库会覆盖当前数据，系统会先自动保存一份灾备副本。是否继续？', positiveText: '确认恢复', negativeText: '取消', onPositiveClick: async () => { try { await restoreDatabase(); message.success('数据库已恢复，请重启应用以刷新所有连接状态。'); await loadSettings() } catch (error) { message.error(String(error)) } } }) }

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
          <NRadioButton v-for="option in themePreferenceOptions" :key="option.value" :value="option.value">{{
            option.label }}</NRadioButton>
        </NRadioGroup>
      </NCard>
      <NCard title="数据库状态" size="medium">
        <p class="app-copy">{{ dbStatus }}</p>
        <p class="app-copy mt-3 break-all">{{ dashboard?.databasePath ?? '等待连接' }}</p>
      </NCard>
      <NCard title="数据安全" size="medium" class="xl:col-span-2">
        <div class="grid gap-3 md:grid-cols-2">
          <NButton size="large" type="primary" @click="handleBackup"><template #icon>
              <NIcon>
                <CloudDownloadOutline />
              </NIcon>
            </template>导出数据备份
            (.db)</NButton>
          <NButton size="large" type="error" ghost @click="handleRestore"><template #icon>
              <NIcon>
                <CloudUploadOutline />
              </NIcon>
            </template>从备份恢复数据</NButton>
        </div>
        <NAlert class="mt-4" type="warning" :show-icon="false"><span class="inline-flex items-center gap-2">
            <NIcon>
              <AlertCircleOutline />
            </NIcon>恢复前会自动生成带时间戳的防灾副本。
          </span></NAlert>
      </NCard>
    </div>
  </section>
</template>
