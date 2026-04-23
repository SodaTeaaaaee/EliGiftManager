<script setup lang="ts">
import { createDiscreteApi, NButton } from 'naive-ui'
import { ref } from 'vue'
import { WAILS_PREVIEW_MESSAGE, isWailsRuntimeAvailable, pingDatabase } from '@/shared/lib/wails/app'

const isPinging = ref(false)
const { message } = createDiscreteApi(['message'])

async function handlePingDB() {
  if (!isWailsRuntimeAvailable()) {
    message.warning(WAILS_PREVIEW_MESSAGE)
    return
  }

  isPinging.value = true

  try {
    const result = await pingDatabase()

    if (result.startsWith('SQLite 读写成功')) {
      message.success(result)
      return
    }

    message.error(result)
  } catch (error) {
    console.error('调用 PingDB 失败', error)
    message.error('调用 PingDB 失败，请查看控制台日志')
  } finally {
    isPinging.value = false
  }
}
</script>

<template>
  <div class="app-viewport flex items-center justify-center bg-stone-950 px-6 py-12">
    <section
      class="w-full max-w-lg rounded-3xl border border-amber-200/10 bg-stone-900/90 p-10 text-center shadow-2xl shadow-black/30"
    >
      <p class="text-sm font-medium uppercase tracking-[0.35em] text-amber-300/80">
        EliGiftManager
      </p>
      <h1 class="mt-4 text-4xl font-semibold tracking-tight text-stone-50">
        SQLite 联调测试
      </h1>
      <p class="mt-4 text-base leading-7 text-stone-300">
        点击下方按钮，前端将通过统一的 Wails 适配层调用 <code>pingDatabase()</code>，
        由 Go 后端完成一次最小化的 SQLite 写入与读取测试。
      </p>

      <NButton
        class="mt-8 w-full"
        type="primary"
        size="large"
        :loading="isPinging"
        @click="handlePingDB"
      >
        测试数据库连通性
      </NButton>
    </section>
  </div>
</template>
