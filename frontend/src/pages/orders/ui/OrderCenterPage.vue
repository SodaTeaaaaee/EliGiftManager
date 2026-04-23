<script setup lang="ts">
const gifts = ['限定徽章套装', '签名明信片', '应援手幅', '周年纪念盒']

const orders = [
  { id: 'EGM-2401', member: 'Mika', gift: '徽章套装', status: '待确认' },
  { id: 'EGM-2402', member: 'Haruka', gift: '明信片', status: '可派发' },
  { id: 'EGM-2403', member: 'Nana', gift: '手幅', status: '地址异常' },
  { id: 'EGM-2404', member: 'Yui', gift: '纪念盒', status: '已锁定' },
]
</script>

<template>
  <section class="space-y-6">
    <header>
      <p class="text-sm font-medium uppercase tracking-[0.24em] text-orange-300">Order Center</p>
      <h1 class="mt-2 text-3xl font-semibold tracking-tight text-neutral-50">派发中心</h1>
      <p class="mt-2 text-neutral-400">左侧选择礼物批次，右侧筛选并处理待派发订单。</p>
    </header>

    <div class="grid min-h-[620px] gap-5 xl:grid-cols-[320px_1fr]">
      <aside class="rounded-3xl border border-white/10 bg-neutral-900/70 p-5">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-neutral-50">礼物列表</h2>
          <span class="text-sm text-neutral-500">4 batches</span>
        </div>
        <div class="mt-5 space-y-3">
          <button
            v-for="(gift, index) in gifts"
            :key="gift"
            class="w-full rounded-2xl border border-white/10 bg-white/[0.04] p-4 text-left transition hover:border-orange-300/50"
          >
            <span class="block text-sm text-neutral-400">Batch {{ index + 1 }}</span>
            <span class="mt-1 block font-medium text-neutral-100">{{ gift }}</span>
            <span class="mt-3 block h-2 rounded-full bg-neutral-700">
              <span class="block h-2 rounded-full bg-orange-300" :style="{ width: `${82 - index * 14}%` }" />
            </span>
          </button>
        </div>
      </aside>

      <section class="rounded-3xl border border-white/10 bg-neutral-900/70 p-5">
        <div class="grid gap-3 md:grid-cols-[1fr_160px_140px]">
          <input
            class="rounded-2xl border border-white/10 bg-neutral-950/60 px-4 py-3 text-sm text-neutral-100 outline-none placeholder:text-neutral-500"
            placeholder="搜索订单号 / 粉丝名"
          />
          <select class="rounded-2xl border border-white/10 bg-neutral-950/60 px-4 py-3 text-sm text-neutral-100 outline-none">
            <option>全部状态</option>
          </select>
          <button class="rounded-2xl bg-orange-400 px-4 py-3 text-sm font-semibold text-neutral-950">筛选</button>
        </div>

        <div class="mt-5 overflow-hidden rounded-2xl border border-white/10">
          <table class="w-full text-left text-sm">
            <thead class="bg-white/[0.04] text-neutral-400">
              <tr>
                <th class="px-4 py-3 font-medium">订单号</th>
                <th class="px-4 py-3 font-medium">粉丝</th>
                <th class="px-4 py-3 font-medium">礼物</th>
                <th class="px-4 py-3 font-medium">状态</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-white/10 text-neutral-200">
              <tr v-for="order in orders" :key="order.id" class="hover:bg-white/[0.03]">
                <td class="px-4 py-4 font-medium text-neutral-100">{{ order.id }}</td>
                <td class="px-4 py-4">{{ order.member }}</td>
                <td class="px-4 py-4">{{ order.gift }}</td>
                <td class="px-4 py-4">
                  <span class="rounded-full bg-orange-300/10 px-3 py-1 text-orange-100">{{ order.status }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </section>
</template>
