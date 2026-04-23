<script setup lang="ts">
const stats = [
  { label: '待派发订单', value: '128', detail: '+18 today' },
  { label: '待补全地址', value: '24', detail: 'needs review' },
  { label: '低库存礼物', value: '7', detail: 'below safety line' },
  { label: '本周已完成', value: '386', detail: '92% fulfilled' },
]

const warnings = [
  '签名明信片库存低于 20 件',
  '3 个订单缺少手机号码',
  '快递模板需要重新授权',
]

const actions = ['导入新订单', '批量匹配地址', '生成发货单', '导出今日报表']
</script>

<template>
  <section class="space-y-6">
    <header>
      <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Dashboard</p>
      <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">工作台</h1>
      <p class="mt-2 text-neutral-400">把今日的派发状态、风险提醒和常用操作放在一个视图里。</p>
    </header>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <article
        v-for="item in stats"
        :key="item.label"
        class="rounded-3xl border border-white/10 bg-white/[0.04] p-5 shadow-xl shadow-black/10"
      >
        <p class="text-sm text-neutral-400">{{ item.label }}</p>
        <p class="mt-3 text-4xl font-semibold text-neutral-50">{{ item.value }}</p>
        <p class="mt-2 text-sm text-orange-200">{{ item.detail }}</p>
      </article>
    </div>

    <div class="grid gap-5 xl:grid-cols-[1.35fr_1fr]">
      <section class="rounded-3xl border border-white/10 bg-neutral-900/70 p-6 shadow-xl shadow-black/20">
        <div class="flex items-center justify-between gap-4">
          <h2 class="text-xl font-semibold text-neutral-50">异常警告</h2>
          <span class="rounded-full bg-red-400/15 px-3 py-1 text-sm text-red-200">3 alerts</span>
        </div>
        <div class="mt-5 space-y-3">
          <div
            v-for="warning in warnings"
            :key="warning"
            class="rounded-2xl border border-red-300/10 bg-red-400/10 p-4 text-sm text-red-100"
          >
            {{ warning }}
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-white/10 bg-neutral-900/70 p-6 shadow-xl shadow-black/20">
        <h2 class="text-xl font-semibold text-neutral-50">快捷操作</h2>
        <div class="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
          <button
            v-for="action in actions"
            :key="action"
            class="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-left text-sm text-neutral-100 transition hover:border-orange-300/50 hover:bg-orange-300/10"
          >
            {{ action }}
          </button>
        </div>
      </section>
    </div>
  </section>
</template>
